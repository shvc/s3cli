package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws/request"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/spf13/cobra"
)

// BuildDate to record build date
var BuildDate = "2018-08-08 08:08:08"

// Version to record build bersion
var Version = "1.0.3"

// Endpoint default Server URL
var Endpoint = "http://s3test.myshare.io:9090"

// S3Client represent a Client
type S3Client struct {
	// credential file
	credential string
	// profile in credential file
	profile string
	// Server URL
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

func (clt *S3Client) newS3Client() (*s3.S3, error) {
	var cred *credentials.Credentials
	if clt.accessKey != "" {
		cred = credentials.NewStaticCredentials(clt.accessKey, clt.secretKey, "")
	} else if clt.credential != "" {
		cred = credentials.NewSharedCredentials(clt.credential, clt.profile)
	} else if clt.profile != "" {
		cred = credentials.NewSharedCredentials("", clt.profile)
	}
	var logLevel *aws.LogLevelType
	if clt.debug {
		logLevel = aws.LogLevel(aws.LogDebug)
	}
	sess, err := session.NewSession(&aws.Config{
		Credentials:      cred,
		Endpoint:         aws.String(clt.endpoint),
		Region:           aws.String(clt.region),
		LogLevel:         logLevel,
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		log.Fatal("NewSession: ", err)
		return nil, err
	}
	return s3.New(sess), nil
}

func (clt *S3Client) createBucket(bucketName string) {
	svc, err := clt.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}
	cparams := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}
	_, err = svc.CreateBucket(cparams)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("Created bucket %s\n", bucketName)
}

func (clt *S3Client) listBucket() {
	svc, err := clt.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}

	bks, err := svc.ListBuckets(nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("Bucket %v\n", *bks)
}

func (clt *S3Client) deleteBucket(bucket string) {
	if bucket == "" {
		log.Fatal("invalid bucket", bucket)
	}
	svc, err := clt.newS3Client()
	if err != nil {
		log.Fatal("init s3 client", err)
	}
	// Create Object
	_, err = svc.DeleteBucket(
		&s3.DeleteBucketInput{
			Bucket: aws.String(bucket),
		})
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("bucket %s deleted\n", bucket)
	}
}

func (clt *S3Client) putObject(bucket, key, filename string, overwrite bool) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Failed to open file", filename, err)
		os.Exit(1)
	}
	defer file.Close()
	if key == "" {
		key = filepath.Base(filename)
	}
	svc, err := clt.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}
	_, err = svc.PutObject(&s3.PutObjectInput{
		Body:   file,
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		fmt.Printf("Failed to upload Object %s/%s, %s\n", bucket, key, err.Error())
		return
	}
	fmt.Printf("Uploaded Object %s\n", key)
}

func (clt *S3Client) mpuObject(bucket, key, filename string, overwrite bool) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Failed to open file", filename, err)
		os.Exit(1)
	}
	defer file.Close()
	if key == "" {
		key = filepath.Base(filename)
	}

	svc, err := clt.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}

	uploader := s3manager.NewUploaderWithClient(svc)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		fmt.Printf("Failed to upload Object %s/%s, %s\n", bucket, key, err.Error())
		return
	}
	fmt.Printf("Uploaded Object %s\n", key)
}

func (clt *S3Client) listObject(bucket, prefix string) {
	svc, err := clt.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}
	obj, err := svc.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		fmt.Println("Failed to list Object", err)
		return
	}
	fmt.Println(obj)
}

func (clt *S3Client) getObject(bucket, key, oRange, filename string) {
	if filename == "" {
		filename = key
	}
	file, err := os.Create(filename)
	if err != nil {
		log.Printf("Unable to open file %s, %v", filename, err)
		return
	}
	defer file.Close()

	svc, err := clt.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if oRange != "" {
		input.SetRange(fmt.Sprintf("bytes=%s", oRange))
	}
	obj, err := svc.GetObject(input)
	if err != nil {
		fmt.Println("Failed to download Object", err)
		return
	}
	io.Copy(file, obj.Body)
	fmt.Printf("Download Object %s\n", key)
}

func (clt *S3Client) deleteObject(bucket, key string) error {
	svc, err := clt.newS3Client()
	if err != nil {
		return err
	}
	_, err = svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

func (clt *S3Client) deleteObjects(bucket, prefix string) (int64, error) {
	var cnt int64
	svc, err := clt.newS3Client()
	if err != nil {
		return cnt, err
	}
	for {
		objects := make([]*s3.ObjectIdentifier, 0, 1000)
		objs, err := svc.ListObjects(&s3.ListObjectsInput{
			Bucket: aws.String(bucket),
			Prefix: aws.String(prefix),
		})
		if err != nil {
			return cnt, err
		}
		objsCnt := len(objs.Contents)
		if objsCnt == 0 {
			return cnt, nil
		}
		for _, obj := range objs.Contents {
			objects = append(objects, &s3.ObjectIdentifier{Key: obj.Key})
		}
		_, err = svc.DeleteObjects(&s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &s3.Delete{Objects: objects, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return cnt, err
		}
		cnt = cnt + int64(objsCnt)
	}
}

func (clt *S3Client) presignObject(bucket, key string, exp int, put bool) (string, error) {
	svc, err := clt.newS3Client()
	if err != nil {
		return "", err
	}
	var req *request.Request
	if put {
		// presign a PUT URL to upload Object
		req, _ = svc.PutObjectRequest(&s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
	} else {
		req, _ = svc.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
	}
	url, err := req.Presign(time.Duration(exp) * time.Hour)
	if err != nil {
		log.Println("Failed to pprsign Object", err)
	}
	return url, err
}

func main() {
	clt := S3Client{}
	var rootCmd = &cobra.Command{
		Use:     "s3cli",
		Short:   "s3cli client tool",
		Long:    "s3cli client tool for S3 Bucket/Object operation",
		Version: fmt.Sprintf("[%s @ %s]", Version, BuildDate),
	}
	rootCmd.PersistentFlags().BoolVarP(&clt.debug, "debug", "d", false, "print debug log")
	rootCmd.PersistentFlags().StringVarP(&clt.credential, "credential", "c", "", "credentail file")
	rootCmd.PersistentFlags().StringVarP(&clt.profile, "profile", "p", "", "credentail profile")
	rootCmd.PersistentFlags().StringVarP(&clt.endpoint, "endpoint", "e", Endpoint, "endpoint")
	rootCmd.PersistentFlags().StringVarP(&clt.accessKey, "accessKey", "a", "", "accessKey")
	rootCmd.PersistentFlags().StringVarP(&clt.secretKey, "secretKey", "s", "", "secretKey")
	rootCmd.PersistentFlags().StringVarP(&clt.region, "region", "g", "cn-north-1", "region")
	rootCmd.Flags().BoolP("version", "v", false, "print version")

	createBucketCmd := &cobra.Command{
		Use:     "createBucket <name>",
		Aliases: []string{"cb"},
		Short:   "create Bucket",
		Long:    "create Bucket",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			clt.createBucket(args[0])
		},
	}
	rootCmd.AddCommand(createBucketCmd)

	listBucketCmd := &cobra.Command{
		Use:     "listBucket",
		Aliases: []string{"lb"},
		Short:   "list Bucket",
		Long:    "list all Buckets",
		Args:    cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			clt.listBucket()
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
			clt.deleteBucket(args[0])
		},
	}
	rootCmd.AddCommand(deleteBucketCmd)

	putObjectCmd := &cobra.Command{
		Use:     "upload <bucket> <local-file>",
		Aliases: []string{"up"},
		Short:   "upload Object",
		Long:    "upload Object to Bucket",
		Args:    cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			key := cmd.Flag("key").Value.String()
			clt.putObject(args[0], key, args[1], cmd.Flag("overwrite").Changed)
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
			clt.mpuObject(args[0], key, args[1], cmd.Flag("overwrite").Changed)
		},
	}
	mpuObjectCmd.Flags().StringP("key", "k", "", "key name")
	mpuObjectCmd.Flags().BoolP("overwrite", "w", false, "overwrite file if exist")
	rootCmd.AddCommand(mpuObjectCmd)

	listObjectCmd := &cobra.Command{
		Use:     "list [bucket] <prefix>",
		Aliases: []string{"ls"},
		Short:   "list Buckets or Objects",
		Long:    "list Buckets or Objects",
		Args:    cobra.RangeArgs(0, 2),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 2 {
				clt.listObject(args[0], args[1])
			} else if len(args) == 1 {
				clt.listObject(args[0], "")
			} else {
				clt.listBucket()
			}
		},
	}
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
			clt.getObject(args[0], args[1], objRange, destination)
		},
	}
	getObjectCmd.Flags().StringP("range", "r", "", "Object range to download, 0-64 means [0, 64]")
	getObjectCmd.Flags().BoolP("overwrite", "w", false, "overwrite file if exist")
	rootCmd.AddCommand(getObjectCmd)

	deleteObjectCmd := &cobra.Command{
		Use:     "delete <bucket> [key]",
		Aliases: []string{"del"},
		Short:   "delete Bucket or Object",
		Long:    "delete Bucket or Object in Bucket",
		Args:    cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 2 {
				if err := clt.deleteObject(args[0], args[1]); err != nil {
					fmt.Println("delete Object error: ", err)
				} else {
					fmt.Println("success")
				}
			} else {
				clt.deleteBucket(args[0])
			}
		},
	}
	rootCmd.AddCommand(deleteObjectCmd)

	deleteObjectsCmd := &cobra.Command{
		Use:     "deleteprefix <bucket> [prefix]",
		Aliases: []string{"dp"},
		Short:   "delete Objects",
		Long:    "delete all Objects with prefix",
		Args:    cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			var cnt int64
			var err error
			if len(args) == 1 {
				cnt, err = clt.deleteObjects(args[0], "")
			} else {
				cnt, err = clt.deleteObjects(args[0], args[1])
			}
			if err != nil {
				fmt.Printf("delete %d Objects meet error: %s\n", cnt, err)
			} else {
				fmt.Printf("delete %d Objects success\n", cnt)
			}
		},
	}
	rootCmd.AddCommand(deleteObjectsCmd)

	presignObjectCmd := &cobra.Command{
		Use:     "presign <bucket> <key>",
		Aliases: []string{"psn", "psg"},
		Short:   "presign Object",
		Long:    "presign Object URL",
		Args:    cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			exp, err := strconv.Atoi(cmd.Flag("hour").Value.String())
			if err != nil {
				fmt.Println("invalid expire time")
				return
			}
			url, err := clt.presignObject(args[0], args[1], exp, cmd.Flag("put").Changed)
			if err != nil {
				fmt.Println("presign failed: ", err)
			} else {
				fmt.Println(url)
			}
		},
	}
	presignObjectCmd.Flags().IntP("hour", "H", 12, "URL expire time(hour)")
	presignObjectCmd.Flags().BoolP("put", "", false, "generate a put URL")
	rootCmd.AddCommand(presignObjectCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
