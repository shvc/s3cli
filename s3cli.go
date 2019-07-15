package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/spf13/cobra"
)

// version to record s3cli version
var version = "1.2.3"

// endpoint ENV Var
var endpointEnvVar = "S3CLI_ENDPOINT"

// S3Cli represent a S3Cli Client
type S3Cli struct {
	profile  string // profile in credential file
	endpoint string // Server endpoine(URL)
	region   string
	verbose  bool
	debug    bool
}

func (sc *S3Cli) loadS3Cfg() (*aws.Config, error) {
	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile(sc.profile))
	if err != nil {
		return nil, fmt.Errorf("failed to load config, %v", err)
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
	return &cfg, nil
}

// newS3Client allocate a s3.Client
func (sc *S3Cli) newS3Client() (*s3.Client, error) {
	cfg, err := sc.loadS3Cfg()
	if err != nil {
		return nil, err
	}
	client := s3.New(*cfg)
	if sc.endpoint == "" {
		sc.endpoint = os.Getenv(endpointEnvVar)
	}
	if sc.endpoint != "" {
		client.ForcePathStyle = true
	}
	return client, nil
}

// listAllObjects list all Objects in spcified bucket
func (sc *S3Cli) listAllObjects(bucket, prefix, delimiter string, index bool) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %v", err)
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
		return fmt.Errorf("list all objects failed: %v", err)
	}
	return nil
}

// listObjects list Objects in spcified bucket
func (sc *S3Cli) listObjects(bucket, prefix, delimiter, marker string, maxkeys int64, index bool) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %v", err)
	}
	req := client.ListObjectsRequest(&s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Marker:    aws.String(marker),
		Delimiter: aws.String(delimiter),
		MaxKeys:   aws.Int64(maxkeys),
	})
	resp, err := req.Send(context.Background())
	if err != nil {
		return fmt.Errorf("list objects failed: %v", err)
	}
	if sc.verbose {
		fmt.Println(resp)
		return nil
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

// getObject downlaod a Object from bucket
func (sc *S3Cli) getObject(bucket, key, oRange, filename string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %v", err)
	}
	// Create a file to write the S3 Object contents to.
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %q, %v", filename, err)
	}
	defer f.Close()
	rangeBytes := ""
	if oRange != "" {
		rangeBytes = fmt.Sprintf("bytes=%s", oRange)
	}
	req := client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Range:  aws.String(rangeBytes),
	})
	resp, err := req.Send(context.Background())
	if err != nil {
		return fmt.Errorf("get object failed: %v", err)
	}
	_, err = io.Copy(f, resp.Body)
	return err
}

// putObject upload a Object
func (sc *S3Cli) putObject(bucket, key, filename string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %v", err)
	}
	// Create a file to write the S3 Object contents to.
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
	_, err = req.Send(context.Background())
	return err
}

func (sc *S3Cli) headObject(bucket, key string) (*s3.HeadObjectOutput, error) {
	client, err := sc.newS3Client()
	if err != nil {
		return nil, fmt.Errorf("init s3 Client failed: %v", err)
	}
	req := client.HeadObjectRequest(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	resp, err := req.Send(context.Background())
	if err != nil {
		return nil, err
	}
	return resp.HeadObjectOutput, nil
}

func (sc *S3Cli) deleteObjects(bucket, prefix string) (int64, error) {
	client, err := sc.newS3Client()
	if err != nil {
		return 0, fmt.Errorf("init s3 Client failed: %v", err)
	}
	var objNum int64
	loi := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}
	doi := &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &s3.Delete{},
	}
	for {
		req := client.ListObjectsRequest(loi)
		resp, err := req.Send(context.Background())
		if err != nil {
			return objNum, fmt.Errorf("list object failed: %v", err)
		}
		contentsLen := len(resp.Contents)
		if contentsLen == 0 {
			break
		}

		objects := make([]s3.ObjectIdentifier, 0, 1000)
		for _, obj := range resp.Contents {
			objects = append(objects, s3.ObjectIdentifier{Key: obj.Key})
		}
		doi.Delete.Objects = objects
		deleteReq := client.DeleteObjectsRequest(doi)
		if _, err = deleteReq.Send(context.Background()); err != nil {
			return objNum, err
		}
		if sc.verbose {
			fmt.Printf("%5d Objects deleted\n", contentsLen)
		}
		objNum += int64(contentsLen)
		if resp.NextMarker != nil {
			loi.Marker = aws.String(*resp.NextMarker)
		} else if resp.IsTruncated != nil && *resp.IsTruncated {
			loi.Marker = resp.Contents[contentsLen-1].Key
		} else {
			break
		}
	}
	return objNum, nil
}

// deleteBucketAndObjects force delete a bucket
func (sc *S3Cli) deleteBucketAndObjects(bucket string) (int64, error) {
	n, err := sc.deleteObjects(bucket, "")
	if err != nil {
		return n, err
	}
	return n, sc.deleteBucket(bucket)
}

func (sc *S3Cli) deleteObject(bucket, key string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %v", err)
	}
	req := client.DeleteObjectRequest(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	_, err = req.Send(context.Background())
	return err
}

func (sc *S3Cli) aclBucket(bucket, key string) error {
	return fmt.Errorf("not ready")
}

func (sc *S3Cli) aclObjects(bucket, prefix string) (int64, error) {
	return 0, fmt.Errorf("not ready")
}

func (sc *S3Cli) aclObject(bucket, key string) error {
	return fmt.Errorf("not ready")
}

// mpuObject Multi-Part-Upload a Object
// TODO: impl
func (sc *S3Cli) mpuObject(bucket, key, filename string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %v", err)
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

// presignGetObject presign a URL to download Object
func (sc *S3Cli) presignGetObject(bucket, key string, exp time.Duration) (string, error) {
	client, err := sc.newS3Client()
	if err != nil {
		return "", fmt.Errorf("init s3 Client failed: %v", err)
	}
	req := client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return req.Presign(exp)
}

// presignPutObject presing a URL to uploda Object
func (sc *S3Cli) presignPutObject(bucket, key string, exp time.Duration) (string, error) {
	client, err := sc.newS3Client()
	if err != nil {
		return "", fmt.Errorf("init s3 Client failed: %v", err)
	}
	req := client.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return req.Presign(exp)
}

func (sc *S3Cli) getObjectACL(bucket, key string) (*s3.GetObjectAclOutput, error) {
	client, err := sc.newS3Client()
	if err != nil {
		return nil, fmt.Errorf("init s3 Client failed: %v", err)
	}
	req := client.GetObjectAclRequest(&s3.GetObjectAclInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	resp, err := req.Send(context.Background())
	if err != nil {
		return nil, err
	}
	return resp.GetObjectAclOutput, nil
}

func (sc *S3Cli) createBucket(bucket string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %v", err)
	}
	createBucketReq := client.CreateBucketRequest(&s3.CreateBucketInput{
		Bucket: aws.String(bucket),
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{
			LocationConstraint: s3.BucketLocationConstraint(sc.region),
		},
	})
	_, err = createBucketReq.Send(context.Background())
	return err
}

func (sc *S3Cli) getBucketACL(bucket string) (*s3.GetBucketAclOutput, error) {
	client, err := sc.newS3Client()
	if err != nil {
		return nil, fmt.Errorf("init s3 Client failed: %v", err)
	}
	req := client.GetBucketAclRequest(&s3.GetBucketAclInput{
		Bucket: aws.String(bucket),
	})
	resp, err := req.Send(context.Background())
	if err != nil {
		return nil, err
	}
	return resp.GetBucketAclOutput, nil
}

func (sc *S3Cli) headBucket(bucket string) (*s3.HeadBucketOutput, error) {
	client, err := sc.newS3Client()
	if err != nil {
		return nil, fmt.Errorf("init s3 Client failed: %v", err)
	}
	req := client.HeadBucketRequest(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	resp, err := req.Send(context.Background())
	if err != nil {
		return nil, err
	}
	return resp.HeadBucketOutput, err
}

func (sc *S3Cli) deleteBucket(bucket string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %v", err)
	}
	req := client.DeleteBucketRequest(&s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	})
	_, err = req.Send(context.Background())
	return err
}

func (sc *S3Cli) listBuckets() error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %v", err)
	}
	req := client.ListBucketsRequest(&s3.ListBucketsInput{})
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
		Long: `s3cli client tool for S3 Bucket/Object operation
Endpoint ENV:
		S3CLI_ENDPOINT=http://host:port (only read if flag -e is not set)

Credential ENV:
		AWS_ACCESS_KEY_ID=AK      (only read if flag -p is not set)
	  	AWS_ACCESS_KEY=AK         (only read if AWS_ACCESS_KEY_ID is not set)
	  	AWS_SECRET_ACCESS_KEY=SK  (only read if flag -p is not set)
	  	AWS_SECRET_KEY=SK         (only read if AWS_SECRET_ACCESS_KEY is not set)`,
		Version: version,
	}
	rootCmd.PersistentFlags().BoolVarP(&sc.debug, "debug", "", false, "print debug log")
	rootCmd.PersistentFlags().BoolVarP(&sc.verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&sc.endpoint, "endpoint", "e", "", "S3 endpoint(http://host:port)")
	rootCmd.PersistentFlags().StringVarP(&sc.profile, "profile", "p", "", "profile in credential file")
	rootCmd.PersistentFlags().StringVarP(&sc.region, "region", "R", endpoints.CnNorth1RegionID, "region")

	createBucketCmd := &cobra.Command{
		Use:     "createBucket <name>",
		Aliases: []string{"cb", "mb"},
		Short:   "create(make) Bucket",
		Long:    "create(make) Bucket",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sc.createBucket(args[0])
		},
	}
	rootCmd.AddCommand(createBucketCmd)

	listBucketCmd := &cobra.Command{
		Use:     "listBuckets",
		Aliases: []string{"lb"},
		Short:   "list Buckets",
		Long:    "list all Buckets",
		Args:    cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			sc.listBuckets()
		},
	}
	rootCmd.AddCommand(listBucketCmd)

	deleteBucketCmd := &cobra.Command{
		Use:     "deleteBucket <bucket>",
		Aliases: []string{"db", "rb"},
		Short:   "delete(remove) Bucket",
		Long:    "delete(remove) Bucket",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := sc.deleteBucket(args[0]); err != nil {
				fmt.Printf("delete %s failed: %s\n", args[0], err)
			}
		},
	}
	rootCmd.AddCommand(deleteBucketCmd)

	headCmd := &cobra.Command{
		Use:     "head <bucket> [key]",
		Aliases: []string{"head"},
		Short:   "head Bucket/Object",
		Long:    "get Bucket/Object metadata",
		Args:    cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 2 {
				if h, err := sc.headObject(args[0], args[1]); err != nil {
					fmt.Printf("head %s failed: %s\n", args[1], err)
				} else {
					fmt.Println(h)
				}
			} else {
				if h, err := sc.headBucket(args[0]); err != nil {
					fmt.Printf("head %s failed: %s\n", args[1], err)
				} else {
					fmt.Println(h)
				}
			}
		},
	}
	rootCmd.AddCommand(headCmd)

	getaclCmd := &cobra.Command{
		Use:     "getacl <bucket> [key]",
		Aliases: []string{"ga"},
		Short:   "get Bucket/Object acl",
		Long:    "get Bucket/Object ACL",
		Args:    cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 2 {
				if acl, err := sc.getObjectACL(args[0], args[1]); err != nil {
					fmt.Printf("get %s ACL failed: %s\n", args[1], err)
				} else {
					fmt.Println(acl)
				}
			} else {
				if acl, err := sc.getBucketACL(args[0]); err != nil {
					fmt.Printf("get %s ACL failed: %s\n", args[0], err)
				} else {
					fmt.Println(acl)
				}
			}
		},
	}
	rootCmd.AddCommand(getaclCmd)

	putObjectCmd := &cobra.Command{
		Use:     "upload <bucket> <local-file>",
		Aliases: []string{"up"},
		Short:   "upload Object",
		Long:    "upload Object to Bucket",
		Args:    cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			key := cmd.Flag("key").Value.String()
			if key == "" {
				key = filepath.Base(args[1])
			}
			if err := sc.putObject(args[0], key, args[1]); err != nil {
				fmt.Printf("upload %s failed: %s\n", args[1], err)
			} else {
				fmt.Printf("upload %s to %s/%s success\n", args[1], args[0], key)
			}
		},
	}
	putObjectCmd.Flags().StringP("key", "k", "", "key name")
	putObjectCmd.Flags().BoolP("overwrite", "w", false, "overwrite file if exist")
	rootCmd.AddCommand(putObjectCmd)

	mpuObjectCmd := &cobra.Command{
		Use:     "mpu <bucket> <local-file>",
		Aliases: []string{"mp", "mu"},
		Short:   "mpu Object",
		Long:    "mutiPartUpload Object to Bucket",
		Args:    cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			//cmd.Flag("overwrite").Changed
			key := cmd.Flag("key").Value.String()
			if key == "" {
				key = filepath.Base(args[0])
			}
			if err := sc.mpuObject(args[0], key, args[1]); err != nil {
				fmt.Printf("mpu %s failed: %s\n", key, err)
			} else {
				fmt.Printf("mpu %s success\n", key)
			}
		},
	}
	mpuObjectCmd.Flags().StringP("key", "k", "", "key name")
	mpuObjectCmd.Flags().BoolP("overwrite", "w", false, "overwrite file if exist")
	rootCmd.AddCommand(mpuObjectCmd)

	listObjectCmd := &cobra.Command{
		Use:     "list [bucket]",
		Aliases: []string{"ls"},
		Short:   "list Buckets or Objects",
		Long:    "list Buckets or Objects",
		Args:    cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			index := cmd.Flag("index").Changed
			prefix := cmd.Flag("prefix").Value.String()
			delimiter := cmd.Flag("delimiter").Value.String()
			if len(args) == 1 {
				if cmd.Flag("all").Changed {
					if err := sc.listAllObjects(args[0], prefix, delimiter, index); err != nil {
						fmt.Println(err)
					}
				} else {
					maxKeys, err := cmd.Flags().GetInt64("maxkeys")
					if err != nil {
						maxKeys = 1000
					}
					marker := cmd.Flag("marker").Value.String()
					if err := sc.listObjects(args[0], prefix, delimiter, marker, maxKeys, index); err != nil {
						fmt.Println(err)
					}
				}
			} else {
				if err := sc.listBuckets(); err != nil {
					fmt.Println(err)
				}
			}
		},
	}
	listObjectCmd.Flags().StringP("marker", "m", "", "marker")
	listObjectCmd.Flags().Int64P("maxkeys", "M", 1000, "max keys")
	listObjectCmd.Flags().StringP("prefix", "x", "", "only show Object(w) with prefix")
	listObjectCmd.Flags().StringP("delimiter", "d", "", "Object delimiter")
	listObjectCmd.Flags().BoolP("index", "i", false, "show Object index ")
	listObjectCmd.Flags().BoolP("all", "a", false, "list all Objects")
	rootCmd.AddCommand(listObjectCmd)

	getObjectCmd := &cobra.Command{
		Use:     "download <bucket> <key> [destination]",
		Aliases: []string{"get", "down", "d"},
		Short:   "download Object",
		Long:    "downlaod Object from Bucket",
		Args:    cobra.RangeArgs(2, 3),
		Run: func(cmd *cobra.Command, args []string) {
			destination := ""
			if len(args) == 3 {
				destination = args[2]
			} else {
				destination = filepath.Base(args[1])
			}
			objRange := cmd.Flag("range").Value.String()
			if err := sc.getObject(args[0], args[1], objRange, destination); err != nil {
				fmt.Printf("download %s to %s failed: %s\n", args[1], destination, err)
			} else {
				fmt.Printf("download %s to %s\n", args[1], destination)
			}
		},
	}
	getObjectCmd.Flags().StringP("range", "r", "", "Object range to download, 0-64 means [0, 64]")
	getObjectCmd.Flags().BoolP("overwrite", "w", false, "overwrite file if exist")
	rootCmd.AddCommand(getObjectCmd)

	deleteObjectCmd := &cobra.Command{
		Use:     "delete <bucket> [key|prefix]",
		Aliases: []string{"del", "rm"},
		Short:   "delete(remove) Object or Bucket(Bucket and Objects)",
		Long:    "delete(remove) Object or Bucket(Bucket and Objects)",
		Args:    cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			prefix := cmd.Flag("prefix").Changed
			if len(args) == 2 && prefix == false {
				if err := sc.deleteObject(args[0], args[1]); err != nil {
					fmt.Println("delete Object failed: ", err)
				}
			} else if prefix {
				prefix := ""
				if len(args) == 2 {
					prefix = args[1]
				}
				if n, err := sc.deleteObjects(args[0], prefix); err != nil {
					fmt.Println("delete Objects failed: ", err)
				} else {
					fmt.Printf("all %d Objects deleted\n", n)
				}
			} else {
				if n, err := sc.deleteBucketAndObjects(args[0]); err != nil {
					fmt.Printf("%d Objects deleted but delete Bucket %s failed: %s\n", n, args[0], err)
				} else {
					fmt.Printf("all %d Objects and Bucket %s deleted\n", n, args[0])
				}
			}
		},
	}
	deleteObjectCmd.Flags().BoolP("prefix", "x", false, "delete Objects start with specified prefix(key)")
	rootCmd.AddCommand(deleteObjectCmd)

	presignObjectCmd := &cobra.Command{
		Use:     "presign <bucket> <key>",
		Aliases: []string{"psn", "psg"},
		Short:   "presign Object",
		Long:    "presign Object URL",
		Args:    cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			exp, err := time.ParseDuration(cmd.Flag("expire").Value.String())
			if err != nil {
				fmt.Println("invalid expire : ", err)
				return
			}
			var url string
			if cmd.Flag("put").Changed {
				url, err = sc.presignPutObject(args[0], args[1], exp)
			} else {
				url, err = sc.presignGetObject(args[0], args[1], exp)
			}
			if err != nil {
				fmt.Println("presign failed: ", err)
			} else {
				fmt.Println(url)
			}
		},
	}
	presignObjectCmd.Flags().DurationP("expire", "E", 12*time.Hour, "URL expire time")
	presignObjectCmd.Flags().BoolP("put", "", false, "generate a put URL")
	rootCmd.AddCommand(presignObjectCmd)

	aclObjectCmd := &cobra.Command{
		Use:     "acl <bucket> [key|prefix]",
		Aliases: []string{"pa"},
		Short:   "acl Bucket or Object",
		Long:    "acl Bucket or Object(s) in Bucket",
		Args:    cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			//prefix := cmd.Flag("prefix").Changed
			key := ""
			if len(args) == 2 {
				key = args[1]
			}
			if cnt, err := sc.aclObjects(args[0], key); err != nil {
				fmt.Println("acl Object error: ", err)
			} else {
				fmt.Printf("acl %d Objects success\n", cnt)
			}

		},
	}
	aclObjectCmd.Flags().BoolP("prefix", "x", false, "acl all Objects with specified prefix(key)")
	rootCmd.AddCommand(aclObjectCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
