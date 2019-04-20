package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"

	"github.com/spf13/cobra"
)

// buildDate to record build date
var buildDate = "2018-08-08 08:08:08"

// version to record build bersion
var version = "1.0.3"

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
	// accessKey(username)
	accessKey string
	// secretKey(password)
	secretKey string
	// debug log
	debug bool
	// region
	region string
	useSSL bool
}

func (sc *S3Cli) loadS3Cfg() (*aws.Config, error) {
	//external.LoadSharedConfig(external.WithSharedConfigProfile(sc.profile))
	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile(sc.profile))
	if err != nil {
		return nil, fmt.Errorf("failed to load config, %v", err)
	}
	cfg.Region = sc.region
	cfg.EndpointResolver = aws.ResolveWithEndpoint{
		URL: sc.endpoint,
	}
	return &cfg, nil
}

func (sc *S3Cli) listObject(bucket, prefix, delimiter string) error {
	cfg, err := sc.loadS3Cfg()
	if err != nil {
		return err
	}
	fmt.Printf("cfg: %v\n\n\n", *cfg)
	svc := s3.New(*cfg)
	req := svc.ListObjectsRequest(&s3.ListObjectsInput{Bucket: &bucket})
	p := req.Paginate()
	for p.Next(context.TODO()) {
		page := p.CurrentPage()
		for _, obj := range page.Contents {
			fmt.Println("Object: ", *obj.Key)
		}
	}

	if err := p.Err(); err != nil {
		return fmt.Errorf("failed to list objects, %v", err)
	}

	return nil
}

func (sc *S3Cli) getObject(bucket, key, oRange, filename string) error {
	cfg, err := sc.loadS3Cfg()
	if err != nil {
		return err
	}

	// Create a downloader with the config and default options
	downloader := s3manager.NewDownloader(*cfg)

	// Create a file to write the S3 Object contents to.
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %q, %v", filename, err)
	}

	// Write the contents of S3 Object to the file
	n, err := downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}
	fmt.Printf("file downloaded, %d bytes\n", n)
	return nil
}

func (sc *S3Cli) putObject(bucket, key, filename string, overwrite bool) error {
	cfg, err := sc.loadS3Cfg()
	if err != nil {
		return err
	}

	// Create an uploader with the config and default options
	uploader := s3manager.NewUploader(*cfg)

	if filename == "" {
		filename = key
	}
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", filename, err)
	}

	// Upload the file to S3.
	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}
	fmt.Printf("file uploaded to, %s\n", result.Location)
	return nil
}

func (sc *S3Cli) headObject(bucket, key string) {

}

func (sc *S3Cli) deleteObjects(bucket, key string, prefix bool) (int, error) {
	return 0, nil
}

func (sc *S3Cli) aclObjects(bucket, key string, prefix bool) (int, error) {
	return 0, nil
}

func (sc *S3Cli) mpuObject(bucket, key, filename string, overwrite bool) {

}

func (sc *S3Cli) presignObject(bucket, key string, exp time.Duration, put bool) (string, error) {

	return "", nil
}

func (sc *S3Cli) getObjectACL(bucket, key string) {

}

func (sc *S3Cli) createBucket(bucket string) {

}

func (sc *S3Cli) getBucketACL(bucket string) {

}

func (sc *S3Cli) headBucket(bucket string) {

}

func (sc *S3Cli) deleteBucket(bucket string) {

}

func (sc *S3Cli) listBucket() {

}

func main() {
	sc := S3Cli{}
	var rootCmd = &cobra.Command{
		Use:     "s3cli",
		Short:   "s3cli client tool",
		Long:    "s3cli client tool for S3 Bucket/Object operation",
		Version: fmt.Sprintf("[%s @ %s]", version, buildDate),
	}
	rootCmd.PersistentFlags().BoolVarP(&sc.debug, "debug", "d", false, "print debug log")
	rootCmd.PersistentFlags().StringVarP(&sc.credential, "credential", "c", "", "credentail file")
	rootCmd.PersistentFlags().StringVarP(&sc.profile, "profile", "p", "", "credentail profile")
	rootCmd.PersistentFlags().StringVarP(&sc.endpoint, "endpoint", "e", endpoint, "endpoint")
	rootCmd.PersistentFlags().StringVarP(&sc.accessKey, "accessKey", "a", "", "accessKey")
	rootCmd.PersistentFlags().StringVarP(&sc.secretKey, "secretKey", "s", "", "secretKey")
	rootCmd.PersistentFlags().StringVarP(&sc.region, "region", "r", "", "region")
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
		Use:     "listBucket",
		Aliases: []string{"lb"},
		Short:   "list Buckets",
		Long:    "list all Buckets",
		Args:    cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			sc.listBucket()
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
			sc.deleteBucket(args[0])
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
				sc.headObject(args[0], args[1])
			} else {
				sc.headBucket(args[0])
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
				sc.getObjectACL(args[0], args[1])
			} else {
				sc.getBucketACL(args[0])
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
			sc.putObject(args[0], key, args[1], cmd.Flag("overwrite").Changed)
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
			key := cmd.Flag("key").Value.String()
			sc.mpuObject(args[0], key, args[1], cmd.Flag("overwrite").Changed)
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
				sc.listObject(args[0], prefix, delimiter)
			} else {
				sc.listBucket()
			}
		},
	}
	listObjectCmd.Flags().StringP("prefix", "P", "", "Object prefix")
	listObjectCmd.Flags().StringP("delimiter", "", "", "Object delimiter")
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
			}
			objRange := cmd.Flag("range").Value.String()
			sc.getObject(args[0], args[1], objRange, destination)
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
			if len(args) == 2 {
				if cnt, err := sc.deleteObjects(args[0], args[1], prefix); err != nil {
					fmt.Println("delete Object error: ", err)
				} else {
					fmt.Printf("deleted %d Objects\n", cnt)
				}
			} else if prefix {
				if cnt, err := sc.deleteObjects(args[0], "", prefix); err != nil {
					fmt.Println("delete Object error: ", err)
				} else {
					fmt.Printf("deleted %d Objects\n", cnt)
				}
			} else {
				sc.deleteBucket(args[0])
			}
		},
	}
	deleteObjectCmd.Flags().BoolP("prefix", "P", false, "delete all Objects with specified prefix(key)")
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
			url, err := sc.presignObject(args[0], args[1], exp, cmd.Flag("put").Changed)
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
			prefix := cmd.Flag("prefix").Changed
			key := ""
			if len(args) == 2 {
				key = args[1]
			}
			if cnt, err := sc.aclObjects(args[0], key, prefix); err != nil {
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
