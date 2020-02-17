package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/spf13/cobra"
)

var (
	// version to record s3cli version
	version = "1.2.3"
	// endpoint ENV Var
	endpointEnvVar = "S3_ENDPOINT"
)

func splitBucketObject(bucketObject string) (bucket, object string) {
	bo := strings.SplitN(bucketObject, "/", 2)
	if len(bo) == 2 {
		return bo[0], bo[1]
	}
	return bucketObject, ""
}

// S3Cli represent a S3Cli Client
type S3Cli struct {
	profile    string // profile in credentials file
	endpoint   string // Server endpoine(URL)
	ak         string // access-key
	sk         string // secret-key
	region     string
	presign    bool // just presign
	presignExp time.Duration
	verbose    bool
	debug      bool
}

// newS3Client allocates a s3.Client
func (sc *S3Cli) newS3Client() (*s3.Client, error) {
	if sc.ak != "" && sc.sk != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", sc.ak)
		os.Setenv("AWS_SECRET_ACCESS_KEY", sc.sk)
	}
	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile(sc.profile))
	if err != nil {
		return nil, fmt.Errorf("failed to load config, %w", err)
	}
	cfg.Region = sc.region
	//cfg.EndpointResolver = aws.ResolveWithEndpoint{
	//	URL: sc.endpoint,
	//}
	defaultResolver := endpoints.NewDefaultResolver()
	myCustomResolver := func(service, region string) (aws.Endpoint, error) {
		if service == s3.EndpointsID {
			return aws.Endpoint{
				URL: sc.endpoint,
				//SigningRegion: "custom-signing-region",
				SigningNameDerived: true,
			}, nil
		}
		return defaultResolver.ResolveEndpoint(service, region)
	}
	cfg.EndpointResolver = aws.EndpointResolverFunc(myCustomResolver)
	if sc.debug {
		cfg.LogLevel = aws.LogDebug
	}
	client := s3.New(cfg)
	if sc.endpoint == "" {
		sc.endpoint = os.Getenv(endpointEnvVar)
	}
	if sc.endpoint != "" {
		client.ForcePathStyle = true
	}
	return client, nil
}

// listAllObjects list all Objects in specified bucket
func (sc *S3Cli) listAllObjects(bucket, prefix, delimiter string, index bool) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	var i int64
	req := client.ListObjectsRequest(&s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delimiter),
	})
	p := s3.NewListObjectsPaginator(req)
	for p.Next(context.TODO()) {
		page := p.CurrentPage()
		if sc.verbose {
			fmt.Println(page)
			continue
		}
		for _, obj := range page.Contents {
			if index {
				fmt.Printf("%d\t%s\n", i, *obj.Key)
				i++
			} else {
				fmt.Println(*obj.Key)
			}
		}
	}
	if err := p.Err(); err != nil {
		return fmt.Errorf("list all objects failed: %w", err)
	}
	return nil
}

// listObjects (S3 listBucket)list Objects in specified bucket
func (sc *S3Cli) listObjects(bucket, prefix, delimiter, marker string, maxkeys int64, index bool) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	req := client.ListObjectsRequest(&s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Marker:    aws.String(marker),
		Delimiter: aws.String(delimiter),
		MaxKeys:   aws.Int64(maxkeys),
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	resp, err := req.Send(context.Background())
	if err != nil {
		return fmt.Errorf("list objects failed: %w", err)
	}
	if sc.verbose {
		fmt.Println(resp)
		return nil
	}
	for _, p := range resp.CommonPrefixes {
		fmt.Println(*p.Prefix)
	}
	for i, obj := range resp.Contents {
		if index {
			fmt.Printf("%d\t%s\n", i, *obj.Key)
		} else {
			fmt.Println(*obj.Key)
		}
	}
	return nil
}

// renameObjects rename Object(s)
func (sc *S3Cli) renameObjects(bucket, prefix, delimiter, marker string) error {
	// TODO: Copy and Delete Object
	return fmt.Errorf("not impl")

}

// copyObjects copy Object to destBucket/key
func (sc *S3Cli) copyObject(source, bucket, key string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	req := client.CopyObjectRequest(&s3.CopyObjectInput{
		CopySource: aws.String(source),
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	resp, err := req.Send(context.Background())
	if err != nil {
		return fmt.Errorf("copy object failed: %w", err)
	}
	if sc.verbose {
		fmt.Println(resp)
		return nil
	}
	return nil
}

// getObject download a Object from bucket
func (sc *S3Cli) getObject(bucket, key, oRange, version, filename string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}

	var objRange *string
	if oRange != "" {
		objRange = aws.String(fmt.Sprintf("bytes=%s", oRange))
	}
	var versionID *string
	if version != "" {
		versionID = aws.String(version)
	}
	req := client.GetObjectRequest(&s3.GetObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String(key),
		VersionId: versionID,
		Range:     objRange,
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	// Create a file to write the S3 Object contents to.
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %q, %w", filename, err)
	}
	defer f.Close()

	resp, err := req.Send(context.Background())
	if err != nil {
		return fmt.Errorf("get object failed: %w", err)
	}
	_, err = io.Copy(f, resp.Body)
	return err
}

// catObject print Object contents
func (sc *S3Cli) catObject(bucket, key, oRange, version string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	var objRange *string
	if oRange != "" {
		objRange = aws.String(fmt.Sprintf("bytes=%s", oRange))
	}
	var versionID *string
	if version != "" {
		versionID = aws.String(version)
	}
	req := client.GetObjectRequest(&s3.GetObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String(key),
		VersionId: versionID,
		Range:     objRange,
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	resp, err := req.Send(context.Background())
	if err != nil {
		return fmt.Errorf("get object failed: %w", err)
	}
	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}

// putObject upload a Object
func (sc *S3Cli) putObject(bucket, key, filename string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}

	if sc.presign {
		req := client.PutObjectRequest(&s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	req := client.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   f,
	})

	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	if sc.verbose {
		fmt.Println(resp)
	}
	return nil
}

func (sc *S3Cli) headObject(bucket, key string, mtime, mtimestamp bool) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	req := client.HeadObjectRequest(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	if sc.verbose {
		fmt.Println(resp.HeadObjectOutput)
	} else if mtime {
		fmt.Println(resp.HeadObjectOutput.LastModified)
	} else if mtimestamp {
		fmt.Println(resp.HeadObjectOutput.LastModified.Unix())
	} else {
		fmt.Printf("%d\t%s\n", *resp.HeadObjectOutput.ContentLength, resp.HeadObjectOutput.LastModified)
	}
	return nil
}

func (sc *S3Cli) getBucketVersioning(bucket string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	req := client.GetBucketVersioningRequest(&s3.GetBucketVersioningInput{
		Bucket: aws.String(bucket),
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("getBucketVersioning: %s\n", resp)
	return nil
}

func (sc *S3Cli) putBucketVersioning(bucket string, status bool) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	verStatus := s3.BucketVersioningStatusSuspended
	if status {
		verStatus = s3.BucketVersioningStatusEnabled
	}
	req := client.PutBucketVersioningRequest(&s3.PutBucketVersioningInput{
		Bucket: aws.String(bucket),
		VersioningConfiguration: &s3.VersioningConfiguration{
			Status: verStatus,
		},
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("putBucketVersioning: %s\n", resp)
	return nil
}

func (sc *S3Cli) listObjectVersions(bucket string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	req := client.ListObjectVersionsRequest(&s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}

	fmt.Println(resp.ListObjectVersionsOutput)
	return nil
}

// deleteObjects list and delete Objects
func (sc *S3Cli) deleteObjects(bucket, prefix string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	var objNum int64
	loi := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}
	for {
		req := client.ListObjectsRequest(loi)
		resp, err := req.Send(context.Background())
		if err != nil {
			return fmt.Errorf("list object failed: %w", err)
		}
		objectNum := len(resp.Contents)
		if objectNum == 0 {
			break
		}
		if sc.verbose {
			fmt.Printf("Got %d Objects, ", objectNum)
		}
		objects := make([]s3.ObjectIdentifier, 0, 1000)
		for _, obj := range resp.Contents {
			objects = append(objects, s3.ObjectIdentifier{Key: obj.Key})
		}
		doi := &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &s3.Delete{Quiet: aws.Bool(true),
				Objects: objects},
		}
		deleteReq := client.DeleteObjectsRequest(doi)
		if _, e := deleteReq.Send(context.Background()); err != nil {
			fmt.Printf("delete Objects failed: %s", e)
		} else {
			objNum = objNum + int64(objectNum)
		}
		if sc.verbose {
			fmt.Printf("%d Objects deleted\n", objNum)
		}

		if resp.NextMarker != nil {
			loi.Marker = resp.NextMarker
		} else if resp.IsTruncated != nil && *resp.IsTruncated {
			loi.Marker = resp.Contents[objectNum-1].Key
		} else {
			break
		}
	}
	return nil
}

// deleteBucketAndObjects force delete a Bucket
func (sc *S3Cli) deleteBucketAndObjects(bucket string, force bool) error {
	if force {
		if err := sc.deleteObjects(bucket, ""); err != nil {
			return err
		}
	}
	return sc.deleteBucket(bucket)
}

func (sc *S3Cli) deleteObject(bucket, key, version string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	var versionID *string
	if version != "" {
		versionID = aws.String(version)
	}
	req := client.DeleteObjectRequest(&s3.DeleteObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String(key),
		VersionId: versionID,
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	if sc.verbose {
		fmt.Println(resp)
	}
	return nil
}

func (sc *S3Cli) policyBucket(bucket string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	req := client.GetBucketPolicyRequest(&s3.GetBucketPolicyInput{
		Bucket: aws.String(bucket),
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	fmt.Println(*resp.GetBucketPolicyOutput.Policy)
	return nil
}

// mpuObject Multi-Part-Upload a Object
// TODO: impl
func (sc *S3Cli) mpuObject(bucket, key, filename string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	// Create a file to write the S3 Object contents to.
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	req := client.CreateMultipartUploadRequest(&s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	req.SetReaderBody(f)
	_, err = req.Send(context.Background())
	return err
}

func (sc *S3Cli) getObjectACL(bucket, key string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}

	req := client.GetObjectAclRequest(&s3.GetObjectAclInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	if resp != nil {
		fmt.Println(resp.GetObjectAclOutput)
	}
	return nil
}

func (sc *S3Cli) createBucket(bucket string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	req := client.CreateBucketRequest(&s3.CreateBucketInput{
		Bucket: aws.String(bucket),
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{
			LocationConstraint: s3.BucketLocationConstraint(sc.region),
		},
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	if sc.verbose {
		fmt.Println(resp)
	}
	return err
}

func (sc *S3Cli) getBucketACL(bucket string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	req := client.GetBucketAclRequest(&s3.GetBucketAclInput{
		Bucket: aws.String(bucket),
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	if resp != nil {
		fmt.Println(resp.GetBucketAclOutput)
	}
	return err
}

func (sc *S3Cli) headBucket(bucket string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	req := client.HeadBucketRequest(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	if resp != nil {
		fmt.Println(resp.HeadBucketOutput)
	}
	return err
}

func (sc *S3Cli) deleteBucket(bucket string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}

	req := client.DeleteBucketRequest(&s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	_, err = req.Send(context.Background())
	return err
}

func (sc *S3Cli) listBuckets() error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	req := client.ListBucketsRequest(&s3.ListBucketsInput{})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	if sc.verbose {
		fmt.Println(resp.ListBucketsOutput)
		return nil
	}
	for _, b := range resp.ListBucketsOutput.Buckets {
		fmt.Println(*b.Name)
	}
	return nil
}

func main() {
	sc := S3Cli{}
	var rootCmd = &cobra.Command{
		Use:   "s3cli",
		Short: "s3cli client tool",
		Long: `S3 commandline tool
Endpoint Envvar:
	S3_ENDPOINT=http://host:port (only read if flag -e is not set)

Credential Envvar:
	AWS_ACCESS_KEY_ID=AK      (only read if flag -p is not set or --ak is not set)
	AWS_ACCESS_KEY=AK         (only read if AWS_ACCESS_KEY_ID is not set)
	AWS_SECRET_ACCESS_KEY=SK  (only read if flag -p is not set or --sk is not set)
	AWS_SECRET_KEY=SK         (only read if AWS_SECRET_ACCESS_KEY is not set)`,
		Version: version,
	}
	rootCmd.PersistentFlags().BoolVarP(&sc.debug, "debug", "", false, "print debug log")
	rootCmd.PersistentFlags().BoolVarP(&sc.verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&sc.presign, "presign", "", false, "presign URL and exit")
	rootCmd.PersistentFlags().DurationVarP(&sc.presignExp, "expire", "", 24*time.Hour, "presign URL expiration")
	rootCmd.PersistentFlags().StringVarP(&sc.endpoint, "endpoint", "e", "", "S3 endpoint(http://host:port)")
	rootCmd.PersistentFlags().StringVarP(&sc.profile, "profile", "p", "", "profile in credentials file")
	rootCmd.PersistentFlags().StringVarP(&sc.region, "region", "R", endpoints.CnNorth1RegionID, "region")
	rootCmd.PersistentFlags().StringVarP(&sc.ak, "ak", "", "", "access key")
	rootCmd.PersistentFlags().StringVarP(&sc.sk, "sk", "", "", "secret key")

	createBucketCmd := &cobra.Command{
		Use:     "makeBucket <bucket>",
		Aliases: []string{"mb"},
		Short:   "make Bucket",
		Long: `make(create) Bucket
1. makeBucket alias
  s3cli mb Bucket`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := sc.createBucket(args[0]); err != nil {
				fmt.Println("makeBucket failed: ", err)
				os.Exit(1)
			}
		},
	}
	rootCmd.AddCommand(createBucketCmd)

	headCmd := &cobra.Command{
		Use:     "head <bucket/key>",
		Aliases: []string{"head"},
		Short:   "head Bucket/Object",
		Long: `get Bucket/Object metadata
1. get a Bucket's Metadata
 s3cli head Bucket
2. get a Object's Metadata
 s3cli head Bucket/Key`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			bucket, key := splitBucketObject(args[0])
			if key != "" {
				mt := cmd.Flag("mtime").Changed
				mts := cmd.Flag("mtimestamp").Changed
				if err := sc.headObject(bucket, key, mt, mts); err != nil {
					fmt.Printf("head %s/%s failed: %s\n", bucket, key, err)
					os.Exit(1)
				}
			} else {
				if err := sc.headBucket(bucket); err != nil {
					fmt.Printf("head %s failed: %s\n", bucket, err)
					os.Exit(1)
				}
			}
		},
	}
	headCmd.Flags().BoolP("mtimestamp", "", false, "show Object mtimestamp")
	headCmd.Flags().BoolP("mtime", "", false, "show Object mtime")
	rootCmd.AddCommand(headCmd)

	aclCmd := &cobra.Command{
		Use:   "acl <bucket/key>",
		Short: "get Bucket/Object ACL",
		Long: `get Bucket/Object ACL
1. get a Bucket's ACL
 s3cli acl Bucket
2. get a Object's ACL
 s3cli acl Bucket/Key`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			bucket, key := splitBucketObject(args[0])
			if key != "" {
				if err := sc.getObjectACL(bucket, key); err != nil {
					fmt.Printf("get %s/%s ACL failed: %s\n", bucket, key, err)
					os.Exit(1)
				}
			} else {
				if err := sc.getBucketACL(bucket); err != nil {
					fmt.Printf("get %s ACL failed: %s\n", bucket, err)
					os.Exit(1)
				}
			}
		},
	}
	rootCmd.AddCommand(aclCmd)

	putObjectCmd := &cobra.Command{
		Use:     "upload [local-file] <bucket/key>",
		Aliases: []string{"put", "up", "u"},
		Short:   "upload Object",
		Long: `upload Object to Bucket
1. upload a file
  s3cli up /path/to/file Bucket
2. upload a file to Bucket/Key
  s3cli up /path/to/file Bucket/Key
3. presign a PUT Object URL
  s3cli up Bucket/Key`,
		Args: cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			bk := args[0]
			if len(args) > 1 {
				bk = args[1]
			} else {
				sc.presign = true
			}
			bucket, key := splitBucketObject(bk)
			if key == "" {
				key = filepath.Base(args[0])
			}
			if err := sc.putObject(bucket, key, args[0]); err != nil {
				fmt.Printf("upload %s failed: %s\n", args[0], err)
				os.Exit(1)
			}
		},
	}
	rootCmd.AddCommand(putObjectCmd)

	mpuObjectCmd := &cobra.Command{
		Use:     "mpu <local-file> <bucket/key>",
		Aliases: []string{"mp", "mu"},
		Short:   "mpu Object",
		Long: `mutiPartUpload Object to Bucket
1. upload a file
  s3cli up /path/to/file Bucket
2. upload a file Bucket/Key
  s3cli up /path/to/file Bucket/Key`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			bucket, key := splitBucketObject(args[1])
			if key == "" {
				key = filepath.Base(args[0])
			}
			if err := sc.mpuObject(bucket, key, args[0]); err != nil {
				fmt.Printf("mpu %s failed: %s\n", key, err)
				os.Exit(1)
			}
		},
	}
	rootCmd.AddCommand(mpuObjectCmd)

	listObjectCmd := &cobra.Command{
		Use:     "list [bucket[/prefix]]",
		Aliases: []string{"ls"},
		Short:   "list Buckets or Objects",
		Long: `list Buckets or Objects
1. list Buckets
  s3cli ls
2. list Objects
  s3cli ls Bucket
3. list Objects with prefix(2019)
  s3cli ls Bucket/2019`,
		Args: cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			index := cmd.Flag("index").Changed
			delimiter := cmd.Flag("delimiter").Value.String()
			if len(args) == 1 {
				bucket, prefix := splitBucketObject(args[0])
				if cmd.Flag("all").Changed {
					if err := sc.listAllObjects(bucket, prefix, delimiter, index); err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
				} else {
					maxKeys, err := cmd.Flags().GetInt64("maxkeys")
					if err != nil {
						maxKeys = 1000
					}
					marker := cmd.Flag("marker").Value.String()
					if err := sc.listObjects(bucket, prefix, delimiter, marker, maxKeys, index); err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
				}
			} else {
				if err := sc.listBuckets(); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			}
		},
	}
	listObjectCmd.Flags().StringP("marker", "m", "", "marker")
	listObjectCmd.Flags().Int64P("maxkeys", "M", 1000, "max keys")
	listObjectCmd.Flags().StringP("delimiter", "d", "", "Object delimiter")
	listObjectCmd.Flags().BoolP("index", "i", false, "show Object index ")
	listObjectCmd.Flags().BoolP("all", "a", false, "list all Objects")
	rootCmd.AddCommand(listObjectCmd)

	getObjectCmd := &cobra.Command{
		Use:     "download <bucket/key> [destination]",
		Aliases: []string{"get", "down", "d"},
		Short:   "download Object",
		Long: `download Object from Bucket
1. download a Object to PWD
  s3cli down Bucket/Key
2. download a Object to /path/to/file
  s3cli down Bucket/Key /path/to/file`,
		Args: cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			bucket, key := splitBucketObject(args[0])
			destination := ""
			if len(args) == 2 {
				destination = args[1]
			} else {
				destination = filepath.Base(key)
			}
			objRange := cmd.Flag("range").Value.String()
			version := cmd.Flag("version").Value.String()
			if err := sc.getObject(bucket, key, objRange, version, destination); err != nil {
				fmt.Printf("download %s to %s failed: %s\n", args[0], destination, err)
				os.Exit(1)
			}
		},
	}
	getObjectCmd.Flags().StringP("range", "r", "", "Object range to download, 0-64 means [0, 64]")
	getObjectCmd.Flags().StringP("version", "", "", "Object version ID to delete")
	getObjectCmd.Flags().BoolP("overwrite", "w", false, "overwrite file if exist")
	rootCmd.AddCommand(getObjectCmd)

	bucketVersionCmd := &cobra.Command{
		Use:     "bucketVersion <bucket>",
		Aliases: []string{"bv"},
		Short:   "bucket versioning",
		Long: `list Object from Bucket
1. get bucket versioning status
  s3cli bv Bucket
2. get bucket versioning status
  s3cli bv Bucket
3. get bucket versioning status
  s3cli bv Bucket
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch cmd.Flag("status").Value.String() {
			case "enable":
				if err := sc.putBucketVersioning(args[0], true); err != nil {
					fmt.Println("enable bucketVersioning failed: ", err)
					os.Exit(1)
				}
			case "disable":
				if err := sc.putBucketVersioning(args[0], false); err != nil {
					fmt.Println("disable bucketVersioning failed: ", err)
					os.Exit(1)
				}
			case "":
				if err := sc.getBucketVersioning(args[0]); err != nil {
					fmt.Printf("listObjectVersions failed: %s\n", err)
					os.Exit(1)
				}
			default:
				fmt.Println("invalid bucketVersioning status")
				os.Exit(1)
			}
		},
	}
	bucketVersionCmd.Flags().StringP("status", "", "", "Set bucketVersioning status(enable, disable)")
	rootCmd.AddCommand(bucketVersionCmd)

	listObjectVersionCmd := &cobra.Command{
		Use:     "listObjectVersion <bucket>",
		Aliases: []string{"lov"},
		Short:   "list Object versions",
		Long: `list Object from Bucket
1. listObjectVersion
  s3cli lov Bucket
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := sc.listObjectVersions(args[0]); err != nil {
				fmt.Printf("listObjectVersions failed: %s\n", err)
				os.Exit(1)
			}
		},
	}
	rootCmd.AddCommand(listObjectVersionCmd)

	catObjectCmd := &cobra.Command{
		Use:   "cat <bucket/key>",
		Short: "cat Object",
		Long: `cat Object contents
1. cat Object
  s3cli cat Bucket/Key`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			objRange := cmd.Flag("range").Value.String()
			version := cmd.Flag("version").Value.String()
			bucket, key := splitBucketObject(args[0])
			if err := sc.catObject(bucket, key, objRange, version); err != nil {
				fmt.Printf("cat %s failed: %s\n", args[0], err)
				os.Exit(1)
			}
		},
	}
	catObjectCmd.Flags().StringP("range", "r", "", "Object range to cat, 0-64 means [0, 64]")
	catObjectCmd.Flags().StringP("version", "", "", "Object version ID to delete")
	rootCmd.AddCommand(catObjectCmd)

	copyObjectCmd := &cobra.Command{
		Use:     "copy <bucket/key> <bucket/key>",
		Aliases: []string{"cp"},
		Short:   "copy Object",
		Long: `copy Bucket/key to Bucket/key
1. spedify destination key
  s3cli copy Bucket1/Key1 Bucket2/Key2
2. default destionation key
  s3cli copy Bucket1/Key1 Bucket2`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			bucket, key := splitBucketObject(args[1])
			if key == "" {
				_, key = splitBucketObject(args[0])
			}
			if err := sc.copyObject(args[0], bucket, key); err != nil {
				fmt.Printf("copy %s failed: %s\n", args[1], err)
				os.Exit(1)
			}
		},
	}
	rootCmd.AddCommand(copyObjectCmd)

	deleteObjectCmd := &cobra.Command{
		Use:     "delete <bucket/key>",
		Aliases: []string{"del", "rm"},
		Short:   "delete(remove) Object or Bucket(Bucket and Objects)",
		Long: `delete Bucket or Object(s)
1. delete Bucket and all Objects
  s3cli delete Bucket
2. delete Object
  s3cli delete Bucket/Key
3. delete all Objects with same Prefix
  s3cli delete Bucket/Prefix -x`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			prefixMode := cmd.Flag("prefix").Changed
			force := cmd.Flag("force").Changed
			bucket, key := splitBucketObject(args[0])
			if prefixMode {
				if err := sc.deleteObjects(bucket, key); err != nil {
					fmt.Println("delete Objects failed: ", err)
					os.Exit(1)
				}
			} else if key != "" {
				version := cmd.Flag("version").Value.String()
				if err := sc.deleteObject(bucket, key, version); err != nil {
					fmt.Println("delete Object failed: ", err)
					os.Exit(1)
				}
			} else {
				if err := sc.deleteBucketAndObjects(bucket, force); err != nil {
					fmt.Printf("deleted Bucket %s and Objects failed: %s\n", args[0], err)
					os.Exit(1)
				}
			}
		},
	}
	deleteObjectCmd.Flags().BoolP("force", "", false, "delete Bucket and all Objects")
	deleteObjectCmd.Flags().StringP("version", "", "", "Object version ID to delete")
	deleteObjectCmd.Flags().BoolP("prefix", "x", false, "delete Objects start with specified prefix")
	rootCmd.AddCommand(deleteObjectCmd)

	policyCmd := &cobra.Command{
		Use:   "policy <bucket>",
		Short: "policy Bucket",
		Long: `policy Bucket
1. get bucket policy
  s3cli policy Bucket
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := sc.policyBucket(args[0]); err != nil {
				fmt.Printf("policy failed: %v\n", err)
				os.Exit(1)
			}
		},
	}
	rootCmd.AddCommand(policyCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
