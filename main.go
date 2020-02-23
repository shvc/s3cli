package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
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

	// MPU command
	mpuCmd := &cobra.Command{
		Use:     "multipartUpload",
		Aliases: []string{"mpu"},
		Short:   "multipart upload",
		Long:    `multipart upload Object`,
	}
	rootCmd.AddCommand(mpuCmd)

	mpuCreateCmd := &cobra.Command{
		Use:     "create <bucket/key>",
		Aliases: []string{"ct"},
		Short:   "create request",
		Long: `create mutiPartUpload
1. create a MPU
	s3cli ct bucket/key
`,
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
		Use:     "upload <file> <bucket/key> <upload-id> <part-num>",
		Aliases: []string{"up", "u"},
		Short:   "upload a MPU part",
		Long: `upload a mutiPartUpload part
1. upload MPU part 1
	s3cli mpu up /path/to/file bucket/key xxxxxx 1
`,
		Args: cobra.ExactArgs(4),
		Run: func(cmd *cobra.Command, args []string) {
			part, err := strconv.ParseInt(args[3], 10, 64)
			if err != nil {
				fmt.Println(err)
				return
			}
			bucket, key := splitBucketObject(args[1])
			if err := sc.mpuUpload(args[0], bucket, key, args[2], part); err != nil {
				fmt.Printf("mpu upload %s failed: %s\n", key, err)
				os.Exit(1)
			}
		},
	}
	mpuCmd.AddCommand(mpuUploadCmd)

	mpuAbortCmd := &cobra.Command{
		Use:     "abort <bucket/key> <upload-id>",
		Aliases: []string{"a"},
		Short:   "abort a MPU",
		Long: `abort mutiPartUpload
1. abort a mpu
	s3cli mpu a bucket/key xxxxxxx
`,
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
1. upload a file
	s3cli mpu ls bucket/prefix
`,
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
		Use:     "complete <bucket/key> <upload-id>",
		Aliases: []string{"cl"},
		Short:   "complete a MPU part",
		Long: `complete mutiPartUpload
1. complete a MPU
	s3cli mpu cl bucket/key xxxxx
`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			bucket, key := splitBucketObject(args[0])
			if err := sc.mpuComplete(bucket, key, args[1]); err != nil {
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
