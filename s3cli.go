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

// endpoint default Server URL
var endpoint = "http://s3test.myshare.io:9090"

// S3Cli represent a S3 Client
type S3Cli struct {
	// credential file
	credential string
	// profile in credential file
	profile string
	// Server endpoine(URL)
	endpoint string
	// debug log
	debug bool
	// region
	region string
}

func (sc *S3Cli) loadS3Cfg() (*aws.Config, error) {
	//external.LoadSharedConfig(external.WithSharedConfigProfile(sc.profile))
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
	if sc.endpoint != "" {
		client.ForcePathStyle = true
	}
	return client, nil
}

// listAllObjects list all Objects in spcified bucket
func (sc *S3Cli) listAllObjects(bucket, prefix, delimiter string) error {
	//fmt.Printf("bucket: %s, prefix: %s, delimiter: %s\n", bucket, prefix, delimiter)
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
		for _, obj := range page.Contents {
			i++
			fmt.Printf("index: %6d, size: %11d, key: %s\n", i, *obj.Size, *obj.Key)
		}
	}
	if err := p.Err(); err != nil {
		return fmt.Errorf("list all objects failed: %v", err)
	}
	return nil
}

// listObjects list all Object in spcified bucket
func (sc *S3Cli) listObjects(bucket, prefix, delimiter string, maxkeys int64) error {
	//fmt.Printf("bucket: %s, prefix: %s, delimiter: %s\n", bucket, prefix, delimiter)
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %v", err)
	}
	req := client.ListObjectsRequest(&s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delimiter),
		MaxKeys:   aws.Int64(maxkeys),
	})
	resp, err := req.Send(context.Background())
	if err != nil {
		return fmt.Errorf("list object failed: %v", err)
	}
	for _, obj := range resp.Contents {
		fmt.Printf("size: %11d, key: %s\n", *obj.Size, *obj.Key)
	}
	return nil
}

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
		fmt.Printf("%d Objects deleted\n", contentsLen)
		objNum += int64(contentsLen)
		if resp.NextMarker != nil {
			loi.Marker = aws.String(*resp.NextMarker)
		} else {
			break
		}
	}
	return objNum, nil
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

func (sc *S3Cli) listBuckets() {
	client, err := sc.newS3Client()
	if err != nil {
		fmt.Printf("init s3 Client failed: %v\n", err)
		return
	}
	req := client.ListBucketsRequest(&s3.ListBucketsInput{})
	if resp, err := req.Send(context.Background()); err != nil {
		fmt.Printf("list buckets failed: %s\n", err)
	} else {
		fmt.Println(resp.ListBucketsOutput)
	}
	return
}

func main() {
	sc := S3Cli{}
	var rootCmd = &cobra.Command{
		Use:     "s3cli",
		Short:   "s3cli client tool",
		Long:    "s3cli client tool for S3 Bucket/Object operation",
		Version: version,
	}
	rootCmd.PersistentFlags().BoolVarP(&sc.debug, "debug", "d", false, "print debug log")
	rootCmd.PersistentFlags().StringVarP(&sc.credential, "credential", "c", "", "credentail file")
	rootCmd.PersistentFlags().StringVarP(&sc.profile, "profile", "p", "", "credentail profile")
	rootCmd.PersistentFlags().StringVarP(&sc.endpoint, "endpoint", "e", endpoint, "endpoint")
	rootCmd.PersistentFlags().StringVarP(&sc.region, "region", "R", endpoints.CnNorth1RegionID, "region")
	rootCmd.Flags().BoolP("version", "v", false, "print version")

	createBucketCmd := &cobra.Command{
		Use:     "createBucket <name>",
		Aliases: []string{"cb"},
		Short:   "create Bucket",
		Long:    "create Bucket",
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
		Aliases: []string{"db"},
		Short:   "delete bucket",
		Long:    "delete a bucket",
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
		Short:   "list Buckets or Objects in Bucket",
		Long:    "list Buckets or Objects in Bucket",
		Args:    cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			prefix := cmd.Flag("prefix").Value.String()
			delimiter := cmd.Flag("delimiter").Value.String()
			if len(args) == 1 {
				var err error
				if cmd.Flag("all").Changed {
					err = sc.listAllObjects(args[0], prefix, delimiter)
				} else {
					maxKeys, err := cmd.Flags().GetInt64("maxkeys")
					if err != nil {
						maxKeys = 1000
					}
					err = sc.listObjects(args[0], prefix, delimiter, maxKeys)
				}
				if err != nil {
					fmt.Println(err)
				}
			} else {
				sc.listBuckets()
			}
		},
	}
	listObjectCmd.Flags().StringP("prefix", "P", "", "Object prefix")
	listObjectCmd.Flags().StringP("delimiter", "", "", "Object delimiter")
	listObjectCmd.Flags().Int64P("maxkeys", "", 1000, "max keys")
	listObjectCmd.Flags().BoolP("all", "", false, "list all Objects")
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
		Short:   "delete Bucket or Object",
		Long:    "delete Bucket or Object(s) in Bucket",
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
				if cnt, err := sc.deleteObjects(args[0], prefix); err != nil {
					fmt.Println("delete Objects failed: ", err)
				} else {
					fmt.Printf("all %d Objects deleted\n", cnt)
				}
			} else {
				if err := sc.deleteBucket(args[0]); err != nil {
					fmt.Printf("delete bucket %s failed: %s\n", args[0], err)
				}
			}
		},
	}
	deleteObjectCmd.Flags().BoolP("prefix", "P", false, "delete Objects start with specified prefix(key)")
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
	aclObjectCmd.Flags().BoolP("prefix", "P", false, "acl all Objects with specified prefix(key)")
	rootCmd.AddCommand(aclObjectCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
