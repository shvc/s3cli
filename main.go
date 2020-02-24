package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

func newS3Client(sc *S3Cli) (*s3.Client, error) {
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
		Hidden:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// mannual init S3 client
			client, err := newS3Client(&sc)
			if err != nil {
				return err
			}
			sc.Client = client
			return nil
		},
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

	// bucket command
	bucketCmd := &cobra.Command{
		Use:     "bucket",
		Aliases: []string{"b"},
		Short:   "bucket sub-command",
		Long:    `bucket sub-command`,
	}
	rootCmd.AddCommand(bucketCmd)

	// bucket sub-command create
	bucketCreateCmd := &cobra.Command{
		Use:     "create <bucket> [<bucket> ...]",
		Aliases: []string{"c"},
		Short:   "create Bucket(s)",
		Long: `create Bucket(s)
* create a Bucket(bk0)
	s3cli b c bk0
* create 3 Buckets(bk1, bk2, bk3)
	s3cli b c bk1 bk2 bk3`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := sc.bucketCreate(args); err != nil {
				fmt.Println("create failed: ", err)
				os.Exit(1)
			}
		},
	}
	bucketCmd.AddCommand(bucketCreateCmd)

	// bucket sub-command list
	bucketListCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "list Buckets",
		Long: `list all my Buckets
* list all my Buckets
  s3cli b ls`,
		Args: cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if err := sc.bucketList(); err != nil {
				fmt.Println("list failed: ", err)
				os.Exit(1)
			}
		},
	}
	bucketCmd.AddCommand(bucketListCmd)

	// bucket sub-command head
	bucketHeadCmd := &cobra.Command{
		Use:     "head <bucket>",
		Aliases: []string{"h"},
		Short:   "head Bucket",
		Long: `head Bucket
* head a Bucket(bk0)
	s3cli b h bk0`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := sc.bucketDelete(args[0]); err != nil {
				fmt.Println("head failed: ", err)
				os.Exit(1)
			}
		},
	}
	bucketCmd.AddCommand(bucketHeadCmd)

	// bucket sub-command acl
	bucketACLCmd := &cobra.Command{
		Use:   "acl <bucket> [ACL]",
		Short: "acl Bucket",
		Long: `acl Bucket
* get a Bucket(bk0)'s ACL
	s3cli b p bk0
* set a Bucket(bk0)'s ACL
	s3cli b p bk0`,
		Args: cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 1 {
				if err := sc.bucketACLGet(args[0]); err != nil {
					fmt.Println("get ACL failed: ", err)
					os.Exit(1)
				}
			} else {
				if err := sc.bucketACLSet(args[0], args[1]); err != nil {
					fmt.Println("set ACL failed: ", err)
					os.Exit(1)
				}
			}
		},
	}
	bucketCmd.AddCommand(bucketACLCmd)

	// bucket sub-command policy
	bucketPolicyCmd := &cobra.Command{
		Use:     "policy <bucket> [policy]",
		Aliases: []string{"p"},
		Short:   "policy Bucket",
		Long: `policy Bucket
* get a Bucket(bk0)'s policy
	s3cli b p bk0
* set a Bucket(bk0)'s policy
	s3cli b p bk0 '{policy}'`,
		Args: cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 1 {
				if err := sc.bucketPolicyGet(args[0]); err != nil {
					fmt.Println("get policy failed: ", err)
					os.Exit(1)
				}
			} else {
				if err := sc.bucketPolicySet(args[0], args[1]); err != nil {
					fmt.Println("set policy failed: ", err)
					os.Exit(1)
				}
			}
		},
	}
	bucketCmd.AddCommand(bucketPolicyCmd)

	// bucket sub-command version
	bucketVersionCmd := &cobra.Command{
		Use:     "version <bucket>",
		Aliases: []string{"v"},
		Short:   "bucket versioning",
		Long: `list Object from Bucket
* get Bucket versioning status
	s3cli b v bucket
* enable bucket versioning
	s3cli b v bucket --status enable
* disable Bucket versioning
	s3cli b v bucket --status disable`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch cmd.Flag("status").Value.String() {
			case "enable":
				if err := sc.bucketVersioningSet(args[0], true); err != nil {
					fmt.Println("enable bucketVersioning failed: ", err)
					os.Exit(1)
				}
			case "disable":
				if err := sc.bucketVersioningSet(args[0], false); err != nil {
					fmt.Println("disable bucketVersioning failed: ", err)
					os.Exit(1)
				}
			case "":
				if err := sc.bucketVersioningGet(args[0]); err != nil {
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
	bucketCmd.AddCommand(bucketVersionCmd)

	// bucket sub-command delete
	bucketDeleteCmd := &cobra.Command{
		Use:     "delete <bucket>",
		Aliases: []string{"d"},
		Short:   "delete Bucket",
		Long: `delete Bucket
* delete a Bucket(bk0)
	s3cli b d bk0`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := sc.bucketDelete(args[0]); err != nil {
				fmt.Println("delete failed: ", err)
				os.Exit(1)
			}
		},
	}
	bucketCmd.AddCommand(bucketDeleteCmd)

	// object put(upload)
	putObjectCmd := &cobra.Command{
		Use:     "put <bucket[/key]> [<local-file> ...]",
		Aliases: []string{"put", "upload"},
		Short:   "put Object(s)",
		Long: `upload Object(s) to Bucket
* put(upload) a file
	s3cli put bucket /path/to/file
* put(upload) a file to Bucket/Key
	s3cli put bucket/key /path/to/file
* put(upload) files to Bucket
	s3cli put bucket file1 file2 file3
* put(upload) files to Bucket with common prefix
	s3cli put bucket/prefix file1 file2 file3
* presign a PUT Object URL
	s3cli up bucket/key`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			bucket, key := splitBucketObject(args[0])
			if len(args) < 2 {
				if err := sc.putObject(bucket, key, ""); err != nil {
					fmt.Printf("put Object failed: %s\n", err)
					os.Exit(1)
				}
			} else {
				for _, v := range args[1:] {
					newKey := fmt.Sprintf("%s%s", key, filepath.Base(v))
					if err := sc.putObject(bucket, newKey, v); err != nil {
						fmt.Printf("put Object failed: %s\n", err)
						os.Exit(1)
					}
				}
			}

		},
	}
	rootCmd.AddCommand(putObjectCmd)

	headCmd := &cobra.Command{
		Use:     "head <bucket/key>",
		Aliases: []string{"head"},
		Short:   "head Bucket/Object",
		Long: `get Bucket/Object metadata
* head a Bucket
	s3cli head bucket
* head a Object
	s3cli head bucket/key`,
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
				if err := sc.bucketHead(bucket); err != nil {
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
* get a Bucket's ACL
	s3cli acl bucket
* get a Object's ACL
	s3cli acl bucket/key`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			bucket, key := splitBucketObject(args[0])
			if key != "" {
				if err := sc.getObjectACL(bucket, key); err != nil {
					fmt.Printf("get %s/%s ACL failed: %s\n", bucket, key, err)
					os.Exit(1)
				}
			} else {
				if err := sc.bucketACLGet(bucket); err != nil {
					fmt.Printf("get %s ACL failed: %s\n", bucket, err)
					os.Exit(1)
				}
			}
		},
	}
	rootCmd.AddCommand(aclCmd)

	listObjectCmd := &cobra.Command{
		Use:     "list [bucket[/prefix]]",
		Aliases: []string{"ls"},
		Short:   "list Buckets or Objects",
		Long: `list Buckets or Objects
* list all my Buckets
	s3cli ls
* list Objects
	s3cli ls bucket
* list Objects with prefix(2019)
	s3cli ls bucket/2019`,
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
				if err := sc.bucketList(); err != nil {
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
		Use:     "get <bucket/key> [destination]",
		Aliases: []string{"download"},
		Short:   "get Object",
		Long: `get(download) Object
* get(download) a Object to ./
	s3cli get bucket/key
* get(download) a Object to /path/to/file
	s3cli get bucket/key /path/to/file
* presign a get(download) Object URL
	s3cli get bucket/key --presign`,
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

	listVersionCmd := &cobra.Command{
		Use:     "listVersion <bucket>",
		Aliases: []string{"lv"},
		Short:   "list Object versions",
		Long: `list Object versions
* list Object Version
	s3cli lv Bucket`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := sc.listObjectVersions(args[0]); err != nil {
				fmt.Printf("listObjectVersions failed: %s\n", err)
				os.Exit(1)
			}
		},
	}
	rootCmd.AddCommand(listVersionCmd)

	catObjectCmd := &cobra.Command{
		Use:   "cat <bucket/key>",
		Short: "cat Object",
		Long: `cat Object contents
* cat a Object
	s3cli cat bucket/key`,
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
* spedify destination key
	s3cli copy bucket1/key1 bucket2/key2
* default destionation key
	s3cli copy bucket1/key1 bucket2`,
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
		Short:   "delete Object or Bucket",
		Long: `delete Bucket or Object(s)
* delete Bucket and all Objects
	s3cli delete Bucket
* delete a Object
	s3cli delete bucket/key
* delete all Objects with same Prefix
	s3cli delete bucket/prefix -x`,
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

	// MPU sub-command
	mpuCmd := &cobra.Command{
		Use:   "mpu",
		Short: "mpu sub-command",
		Long:  `mpu sub-command`,
	}
	rootCmd.AddCommand(mpuCmd)

	mpuCreateCmd := &cobra.Command{
		Use:     "create <bucket/key>",
		Aliases: []string{"ct"},
		Short:   "create a MPU request",
		Long: `create a mutiPartUpload request
* create a MPU request
	s3cli mpu ct bucket/key`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			bucket, key := splitBucketObject(args[0])
			if err := sc.mpuCreate(bucket, key); err != nil {
				fmt.Printf("mpu %s failed: %s\n", key, err)
				os.Exit(1)
			}
		},
	}
	mpuCmd.AddCommand(mpuCreateCmd)

	mpuUploadCmd := &cobra.Command{
		Use:     "upload <bucket/key> <upload-id> <part-num> <file>",
		Aliases: []string{"up", "u"},
		Short:   "upload a MPU part",
		Long: `upload a mutiPartUpload part
* upload MPU part 1
	s3cli mpu u bucket/key upload-id 1 /path/to/file`,
		Args: cobra.ExactArgs(4),
		Run: func(cmd *cobra.Command, args []string) {
			part, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				fmt.Println(err)
				return
			}
			bucket, key := splitBucketObject(args[0])
			if err := sc.mpuUpload(bucket, key, args[1], part, args[3]); err != nil {
				fmt.Printf("mpu upload %s failed: %s\n", key, err)
				os.Exit(1)
			}
		},
	}
	mpuCmd.AddCommand(mpuUploadCmd)

	mpuAbortCmd := &cobra.Command{
		Use:     "abort <bucket/key> <upload-id>",
		Aliases: []string{"a"},
		Short:   "abort a MPU request",
		Long: `abort a mutiPartUpload request
1. abort a mpu request
	s3cli mpu a bucket/key upload-id`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			bucket, key := splitBucketObject(args[0])
			if err := sc.mpuAbort(bucket, key, args[1]); err != nil {
				fmt.Printf("mpu abort %s failed: %s\n", args[1], err)
				os.Exit(1)
			}
		},
	}
	mpuCmd.AddCommand(mpuAbortCmd)

	mpuListCmd := &cobra.Command{
		Use:     "list <bucket/prefix>",
		Aliases: []string{"ls"},
		Short:   "list MPU",
		Long: `list mutiPartUploads
1. list MPU
	s3cli mpu ls bucket/prefix`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			bucket, key := splitBucketObject(args[0])
			if err := sc.mpuList(bucket, key); err != nil {
				fmt.Printf("mpu list failed: %s\n", err)
				os.Exit(1)
			}
		},
	}
	mpuCmd.AddCommand(mpuListCmd)

	mpuCompleteCmd := &cobra.Command{
		Use:     "complete <bucket/key> <upload-id> <part-etag> [<part-etag> ...]",
		Aliases: []string{"cl"},
		Short:   "complete a MPU request",
		Long: `complete a mutiPartUpload request
1. complete a MPU request
	s3cli mpu cl bucket/key upload-id etag01 etag02 etag03`,
		Args: cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			bucket, key := splitBucketObject(args[0])
			etags := make([]string, len(args)-2)
			for i := range etags {
				etags[i] = args[i+2]
			}
			if err := sc.mpuComplete(bucket, key, args[1], etags); err != nil {
				fmt.Printf("mpu complete %s failed: %s\n", args[1], err)
				os.Exit(1)
			}
		},
	}
	mpuCmd.AddCommand(mpuCompleteCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
