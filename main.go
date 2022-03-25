package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/corehandlers"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/spf13/cobra"
)

const (
	defaultDialTimeout           = 10
	defaultResponseHeaderTimeout = 20
)

var (
	// version to record s3cli version
	version = "1.2.3"
	// endpoint ENV Var
	endpointEnvVar = "S3_ENDPOINT"
	// With ForcePathStyle(pathStyle=true):
	// 	https://s3.us-west-2.amazonaws.com/BUCKET/KEY
	// Without ForcePathStyle(pathStyle=false):
	// 	https://BUCKET.s3.us-west-2.amazonaws.com/KEY
	pathStyle             = true
	dialTimeout           int
	responseHeaderTimeout int
	httpKeepAlive         = true
	v2Sign                = false
)

func newS3Client(sc *S3Cli) (*s3.S3, error) {
	if sc.endpoint == "" {
		sc.endpoint = os.Getenv(endpointEnvVar)
	}
	if sc.endpoint == "" {
		return nil, errors.New("unknown endpoint")
	}

	if sc.accessKey == "" {
		sc.accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		if sc.accessKey == "" {
			sc.accessKey = os.Getenv("AWS_ACCESS_KEY")
		}
	}

	if sc.secretKey == "" {
		sc.secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		if sc.secretKey == "" {
			sc.secretKey = os.Getenv("AWS_SECRET_KEY")
		}
	}

	if sc.accessKey == "" && sc.secretKey != "" {
		return nil, errors.New("unknown accessKey")
	}

	if sc.accessKey != "" && sc.secretKey == "" {
		return nil, errors.New("unknown secretKey")
	}

	cfg := &aws.Config{
		Region:           aws.String(sc.region),
		MaxRetries:       aws.Int(0),
		S3ForcePathStyle: aws.Bool(pathStyle),
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
				Dial:                  (&net.Dialer{Timeout: time.Duration(dialTimeout) * time.Second}).Dial,
				ResponseHeaderTimeout: time.Duration(responseHeaderTimeout) * time.Second,
				DisableKeepAlives:     !httpKeepAlive,
			},
		},
		EndpointResolver: endpoints.ResolverFunc(
			func(service, region string, opts ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
				return endpoints.ResolvedEndpoint{
					URL:           sc.endpoint,
					SigningRegion: sc.region,
					SigningName:   service,
					SigningMethod: "v4",
				}, nil

			}),
	}
	if sc.profile != "" {
		cfg.Credentials = credentials.NewSharedCredentials("", sc.profile)
	} else if sc.accessKey == "" && sc.secretKey == "" {
		cfg.Credentials = credentials.AnonymousCredentials
	} else {
		cfg.Credentials = credentials.NewStaticCredentials(sc.accessKey, sc.secretKey, "")
	}
	sess := session.Must(session.NewSession(cfg))

	if sc.debug {
		sess.Config.LogLevel = aws.LogLevel(aws.LogDebug)
	}
	svc := s3.New(sess)
	if v2Sign {
		svc.Handlers.Sign.Clear()
		svc.Handlers.Sign.PushBackNamed(corehandlers.BuildContentLengthHandler)
		svc.Handlers.Sign.PushBack(func(req *request.Request) {
			if req.Config.Credentials == credentials.AnonymousCredentials {
				return
			}
			if req.ExpireTime > 0 {
				v2Presign(sc.accessKey, sc.secretKey, req.ExpireTime, req.HTTPRequest)
			} else {
				sign(sc.accessKey, sc.secretKey, req.HTTPRequest)
			}
		})
	}

	return svc, nil
}

func main() {
	sc := S3Cli{}
	objectMetadata := []string{}
	objectContentType := ""
	var rootCmd = &cobra.Command{
		Use:   "s3cli",
		Short: "s3cli",
		Long: `
EnvVar:
	S3_ENDPOINT=http://host:port (only read if flag --endpoint is not set)
	AWS_PROFILE=profile          (only read if flag --profile is not set)
	AWS_ACCESS_KEY_ID=ak         (only read if flag --ak and --profile not set)
	AWS_ACCESS_KEY=ak            (only read if AWS_ACCESS_KEY_ID is not set)
	AWS_SECRET_ACCESS_KEY=sk     (only read if flag --sk and --profile not set)
	AWS_SECRET_KEY=sk            (only read if AWS_SECRET_ACCESS_KEY is not set)`,
		Version: version,
		Hidden:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client, err := newS3Client(&sc)
			if err != nil {
				return sc.errorHandler(err)
			}
			sc.Client = client
			return nil
		},
	}
	rootCmd.PersistentFlags().BoolVarP(&sc.debug, "debug", "", false, "show SDK debug log")
	rootCmd.PersistentFlags().StringVarP(&sc.output, "output", "o", outputSimple, "output format(verbose,simple,json,line)")
	rootCmd.PersistentFlags().BoolVarP(&sc.presign, "presign", "", false, "presign Request and exit")
	rootCmd.PersistentFlags().DurationVarP(&sc.presignExp, "presign-exp", "", 24*time.Hour, "presign Request expiration duration")
	rootCmd.PersistentFlags().StringVarP(&sc.endpoint, "endpoint", "e", "", "S3 endpoint(http://host:port)")
	rootCmd.PersistentFlags().StringVarP(&sc.profile, "profile", "p", "", "profile in credentials file")
	rootCmd.PersistentFlags().StringVarP(&sc.region, "region", "R", s3.BucketLocationConstraintCnNorth1, "S3 region")
	rootCmd.PersistentFlags().StringVarP(&sc.accessKey, "ak", "a", "", "S3 access key(only read if profile not set)")
	rootCmd.PersistentFlags().StringVarP(&sc.secretKey, "sk", "s", "", "S3 secret key(only read if profile not set)")
	rootCmd.PersistentFlags().BoolVarP(&pathStyle, "path-style", "", true, "use path style")
	rootCmd.PersistentFlags().BoolVarP(&httpKeepAlive, "http-keep-alive", "", true, "http Keep-Alive")
	rootCmd.PersistentFlags().BoolVarP(&v2Sign, "v2sign", "", false, "S3 signature v2")
	rootCmd.PersistentFlags().IntVarP(&dialTimeout, "dial-timeout", "", defaultDialTimeout, "http dial timeout in seconds")
	rootCmd.PersistentFlags().IntVarP(&responseHeaderTimeout, "response-header-timeout", "", defaultResponseHeaderTimeout, "http response header timeout in seconds")
	rootCmd.PersistentFlags().StringArrayVarP(&sc.header, "header", "H", nil, "Pass custom header(s) to server(format Key:Value)")
	// presign(V2) command
	presignCmd := &cobra.Command{
		Use:   "presign <bucket/key>",
		Short: "presign(V2 and not escape URL path) a request",
		Long: `presign(V2 and not escape URL path) usage:
* presign a GET Object URL
	s3cli presign bucket-name/key(01)
* presign a DELETE Object URL
	s3cli presign -X delete bucket-name/key(01)
* presign a PUT Object URL and specify content-type
	s3cli presign -X PUT --content-type text/plain bucket-name/key(01)
	curl -X PUT -H content-type:text/plain -d test-str 'presign-url'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(cmd.Flag("method").Value.String())
			switch method {
			case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPost, http.MethodDelete:
				break
			default:
				return sc.errorHandler(fmt.Errorf("invalid http method: %s", method))
			}
			var s string
			var err error
			s, err = sc.presignV2Raw(method, args[0], objectContentType)
			if err != nil {
				return sc.errorHandler(err)
			}
			fmt.Println(s)
			return nil
		},
	}
	presignCmd.Flags().StringP("method", "X", http.MethodGet, "http request method")
	presignCmd.Flags().StringVar(&objectContentType, "content-type", "", "http request content-type")
	rootCmd.AddCommand(presignCmd)

	bucketCreateCmd := &cobra.Command{
		Use:     "create-bucket <bucket> [<bucket> ...]",
		Aliases: []string{"cb"},
		Short:   "create Bucket(s)",
		Long: `create Bucket(s) usage:
* create a Bucket
	s3cli create-bucket bucket-name
* create 3 Buckets(bkt1, bkt2 and bkt3)
	s3cli create-bucket bkt1 bkt2 bkt3`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return sc.errorHandler(sc.bucketCreate(args))
		},
	}
	rootCmd.AddCommand(bucketCreateCmd)

	bucketPolicyCmd := &cobra.Command{
		Use:   "policy <bucket> [policy]",
		Short: "get/set Bucket Policy",
		Long: `get/set Bucket Policy usage:
* get Bucket policy
	s3cli policy bucket-name
* set Bucket policy(a json string)
	s3cli policy bucket-name '{json}'`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return sc.errorHandler(sc.bucketPolicyGet(args[0]))
			}
			return sc.errorHandler(sc.bucketPolicySet(args[0], args[1]))
		},
	}
	rootCmd.AddCommand(bucketPolicyCmd)

	bucketVersionCmd := &cobra.Command{
		Use:     "version <bucket/key> [arg]",
		Aliases: []string{"v"},
		Short:   "bucket versioning",
		Long: `get/set bucket versioning status usage:
* get Bucket versioning status
	s3cli version bucket-name
* enable bucket versioning
	s3cli version bucket-name Enabled
* suspend Bucket versioning
	s3cli version bucket-name Suspended
* get Object versions
	s3cli version bucket-name/key
	`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				bucket, prefix := sc.splitKeyValue(args[0], "/")
				if prefix == "" {
					return sc.errorHandler(sc.bucketVersioningGet(bucket))
				}
				return sc.errorHandler(sc.listObjectVersions(bucket, prefix))
			}

			var status string
			switch strings.ToLower(args[1]) {
			case strings.ToLower(s3.BucketVersioningStatusEnabled):
				status = s3.BucketVersioningStatusEnabled
			case strings.ToLower(s3.BucketVersioningStatusSuspended):
				status = s3.BucketVersioningStatusSuspended
			default:
				return sc.errorHandler(fmt.Errorf("invalid versioning: %v", args[1]))
			}
			return sc.errorHandler(sc.bucketVersioningSet(args[0], status))
		},
	}
	rootCmd.AddCommand(bucketVersionCmd)

	var corsDelete bool
	bucketCorsCmd := &cobra.Command{
		Use:   "cors <bucket> [arg]",
		Short: "bucket cors",
		Long: `get/delete/set bucket cors usage:
* get Bucket cors
	s3cli cors bucket-name
* delete Bucket cors
	s3cli cors bucket-name --delete
* set Bucket cors
	s3cli cors bucket-name cors.json
`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			bucket, _ := sc.splitKeyValue(args[0], "/")
			if len(args) == 1 {
				if corsDelete {
					return sc.errorHandler(sc.deleteBucketCors(bucket))
				} else {
					return sc.errorHandler(sc.getBucketCors(bucket))
				}
			} else {
				return sc.errorHandler(sc.putBucketCors(bucket, args[1]))
			}

		},
	}
	bucketCorsCmd.Flags().BoolVar(&corsDelete, "delete", false, "delete bucket cors")
	rootCmd.AddCommand(bucketCorsCmd)

	// object upload(put)
	uploadObjectCmd := &cobra.Command{
		Use:     "upload <bucket[/key]> [file ...]",
		Aliases: []string{"put"},
		Short:   "upload Object(s)",
		Long: `upload Object(s) usage:
* upload a file
	s3cli upload bucket /path/to/file
* upload a file to Bucket/Key
	s3cli upload bucket-name/key /path/to/file
* upload files to Bucket
	s3cli upload bucket-name file1 file2 file3
	s3cli upload bucket-name *.txt
* upload files to Bucket with specified common prefix(dir/)
	s3cli upload bucket-name/dir/ file1 file2 file3
	s3cli upload bucket-name/dir2/ *.txt
* presign(V4) a PUT Object URL
	s3cli upload bucket-name/key --presign`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var fd *os.File
			stream := cmd.Flag("stream").Changed
			bucket, key := sc.splitKeyValue(args[0], "/")
			var metadata map[string]*string
			for _, v := range objectMetadata {
				k, v := sc.splitKeyValue(v, ":")
				if k != "" && v != "" {
					if metadata == nil {
						metadata = make(map[string]*string)
					}
					metadata[k] = &v
				}
			}
			if len(args) < 2 { // upload a zero-size file
				err = sc.putObject(bucket, key, objectContentType, metadata, stream, fd)
			} else if len(args) == 2 { // upload one file
				if key == "" {
					key = filepath.Base(args[1])
				}
				fd, err = os.Open(args[1])
				if err != nil {
					return sc.errorHandler(err)
				}
				defer fd.Close()
				if objectContentType == "" {
					objectContentType = mime.TypeByExtension(filepath.Ext(args[1]))
				}
				err = sc.putObject(bucket, key, objectContentType, metadata, stream, fd)
			} else { // upload multi files
				for _, v := range args[1:] {
					fd, err = os.Open(v)
					if err != nil {
						return sc.errorHandler(err)
					}
					if objectContentType == "" {
						objectContentType = mime.TypeByExtension(filepath.Ext(args[1]))
					}
					newKey := key + filepath.Base(v)
					err = sc.putObject(bucket, newKey, objectContentType, metadata, stream, fd)
					if err != nil {
						fd.Close()
						return sc.errorHandler(err)
					}
					fd.Close()
				}
			}
			return sc.errorHandler(err)
		},
	}
	uploadObjectCmd.Flags().StringVar(&objectContentType, "content-type", "", "Object content-type(auto detect if not specified)")
	uploadObjectCmd.Flags().BoolP("stream", "", false, "stream mode(header Transfer-Encoding: chunked)")
	uploadObjectCmd.Flags().StringArrayVar(&objectMetadata, "md", nil, "Object user metadata(format Key:Value)")
	rootCmd.AddCommand(uploadObjectCmd)

	headCmd := &cobra.Command{
		Use:   "head <bucket/key>",
		Short: "head Bucket or Object",
		Long: `head Bucket or Object usage:
* head a Bucket
	s3cli head bucket-name
* head a Object
	s3cli head bucket-name/key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bucket, key := sc.splitKeyValue(args[0], "/")
			if key != "" {
				mt := cmd.Flag("mtime").Changed
				mts := cmd.Flag("mtimestamp").Changed
				return sc.errorHandler(sc.headObject(bucket, key, mt, mts))
			}
			return sc.errorHandler(sc.bucketHead(bucket))
		},
	}
	headCmd.Flags().BoolP("mtimestamp", "", false, "show Object mtimestamp")
	headCmd.Flags().BoolP("mtime", "", false, "show Object mtime")
	rootCmd.AddCommand(headCmd)

	aclCmd := &cobra.Command{
		Use:   "acl <bucket/key> [ACL]",
		Short: "get/set Bucket/Object ACL",
		Long: `get/set Bucket/Object ACL usage:
* get Bucket ACL
	s3cli acl bucket-name
* set Bucket ACL to public-read
	s3cli acl bucket-name public-read
* get Object ACL
	s3cli acl bucket-name/key
* set Object ACL to public-read
	s3cli acl bucket-name/key public-read

* all canned ACL(private,public-read,public-read-write,authenticated-read,aws-exec-read,bucket-owner-read,bucket-owner-full-control)
`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bucket, key := sc.splitKeyValue(args[0], "/")
			if key != "" { // Object ACL
				if len(args) == 1 {
					return sc.errorHandler(sc.getObjectACL(bucket, key))
				}
				var acl string
				switch args[1] {
				case s3.ObjectCannedACLPrivate:
					acl = s3.ObjectCannedACLPrivate
				case s3.ObjectCannedACLPublicRead:
					acl = s3.ObjectCannedACLPublicRead
				case s3.ObjectCannedACLPublicReadWrite:
					acl = s3.ObjectCannedACLPublicReadWrite
				case s3.ObjectCannedACLAuthenticatedRead:
					acl = s3.ObjectCannedACLAuthenticatedRead
				case s3.ObjectCannedACLAwsExecRead:
					acl = s3.ObjectCannedACLAwsExecRead
				case s3.ObjectCannedACLBucketOwnerRead:
					acl = s3.ObjectCannedACLBucketOwnerRead
				case s3.ObjectCannedACLBucketOwnerFullControl:
					acl = s3.ObjectCannedACLBucketOwnerFullControl
				default:
					return sc.errorHandler(fmt.Errorf("invalid ACL: %s", args[1]))
				}
				return sc.errorHandler(sc.setObjectACL(bucket, key, acl))
			}
			// Bucket ACL
			if len(args) == 1 {
				return sc.errorHandler(sc.bucketACLGet(bucket))
			}
			var acl string
			switch args[1] {
			case s3.BucketCannedACLPrivate:
				acl = s3.BucketCannedACLPrivate
			case s3.BucketCannedACLPublicRead:
				acl = s3.BucketCannedACLPublicRead
			case s3.BucketCannedACLPublicReadWrite:
				acl = s3.BucketCannedACLPublicReadWrite
			case s3.BucketCannedACLAuthenticatedRead:
				acl = s3.BucketCannedACLAuthenticatedRead
			default:
				return sc.errorHandler(fmt.Errorf("invalid ACL: %s", args[1]))
			}
			return sc.errorHandler(sc.bucketACLSet(args[0], acl))
		},
	}
	rootCmd.AddCommand(aclCmd)

	//aws --endpoint-url http://172.16.3.98:9020 --profile ak1 s3api list-buckets
	//aws --endpoint-url http://172.16.3.98:9020 --profile ak1 s3api list-objects --bucket mybucket
	listObjectCmd := &cobra.Command{
		Use:     "list [bucket[/prefix]]",
		Aliases: []string{"ls"},
		Short:   "list Buckets or Objects",
		Long: `list Buckets or Objects usage:
* list all my Buckets
	s3cli ls
* list Objects in a Bucket
	s3cli ls bucket-name
* list Objects with prefix(2019)
	s3cli ls bucket-name/2019
* list Objects(2006-01-02T15:04:05Z < modifyTime < 2020-06-03T00:00:00Z)
	s3cli ls bucket-name --start-time 2006-01-02T15:04:05Z --end-time 2020-06-03T00:00:00Z
* list Objects(2006-01-02T15:04:05Z < modifyTime < 2020-06-03T00:00:00Z) start with common prefix
	s3cli ls bucket-name/prefix --start-time 2006-01-02T15:04:05Z --end-time 2020-06-03T00:00:00Z
`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			index := cmd.Flag("index").Changed
			delimiter := cmd.Flag("delimiter").Value.String()
			if len(args) == 1 { // list Objects
				stime, err := time.Parse("2006-01-02T15:04:05Z", cmd.Flag("start-time").Value.String())
				if err != nil {
					return sc.errorHandler(fmt.Errorf("invalid start-time %s, error %s", cmd.Flag("start-time").Value.String(), err))
				}
				etime, err := time.Parse("2006-01-02T15:04:05Z", cmd.Flag("end-time").Value.String())
				if err != nil {
					return sc.errorHandler(fmt.Errorf("invalid end-time %s, error %s", cmd.Flag("end-time").Value.String(), err))
				}

				bucket, prefix := sc.splitKeyValue(args[0], "/")
				if cmd.Flag("all").Changed {
					return sc.errorHandler(sc.listAllObjects(bucket, prefix, delimiter, index, stime, etime))
				}
				maxKeys, err := cmd.Flags().GetInt64("maxkeys")
				if err != nil {
					maxKeys = 1000
				}
				marker := cmd.Flag("marker").Value.String()
				return sc.errorHandler(sc.listObjects(bucket, prefix, delimiter, marker, maxKeys, index, stime, etime))
			}

			// list all my Buckets
			return sc.errorHandler(sc.bucketList())
		},
	}
	listObjectCmd.Flags().StringP("marker", "m", "", "marker")
	listObjectCmd.Flags().Int64P("maxkeys", "M", 1000, "max keys")
	listObjectCmd.Flags().StringP("delimiter", "d", "", "Object delimiter")
	listObjectCmd.Flags().BoolP("index", "i", false, "show Object index ")
	listObjectCmd.Flags().BoolP("all", "", false, "list all Objects")
	listObjectCmd.Flags().StringP("start-time", "", "2006-01-02T15:04:05Z", "show Objects modify-time after start-time(UTC)")
	listObjectCmd.Flags().StringP("end-time", "", "2060-01-02T15:04:05Z", "show Objects modify-time before end-time(UTC)")
	rootCmd.AddCommand(listObjectCmd)

	listObjectV2Cmd := &cobra.Command{
		Use:     "list-v2 [bucket[/prefix]]",
		Aliases: []string{"lsv2"},
		Short:   "list Buckets or Objects(API V2)",
		Long: `list-v2 Buckets or Objects(API V2) usage:
* list all my Buckets
	s3cli list-v2
* list Objects in a Bucket
	s3cli list-v2 bucket-name
* list Objects with prefix(2019)
	s3cli list-v2 bucket-name/2019
* list Objects(2006-01-02T15:04:05Z < modifyTime < 2020-06-03T00:00:00Z)
	s3cli list-v2 bucket-name --start-time 2006-01-02T15:04:05Z --end-time 2020-06-03T00:00:00Z
* list Objects(2006-01-02T15:04:05Z < modifyTime < 2020-06-03T00:00:00Z) start with common prefix
	s3cli list-v2 bucket-name/prefix --start-time 2006-01-02T15:04:05Z --end-time 2020-06-03T00:00:00Z
`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			index := cmd.Flag("index").Changed
			fetchOwner := cmd.Flag("owner").Changed
			delimiter := cmd.Flag("delimiter").Value.String()
			if len(args) == 1 { // list Objects
				stime, err := time.Parse("2006-01-02T15:04:05Z", cmd.Flag("start-time").Value.String())
				if err != nil {
					return sc.errorHandler(fmt.Errorf("invalid start-time %s, error %s", cmd.Flag("start-time").Value.String(), err))
				}
				etime, err := time.Parse("2006-01-02T15:04:05Z", cmd.Flag("end-time").Value.String())
				if err != nil {
					return sc.errorHandler(fmt.Errorf("invalid enf-time %s, error %s", cmd.Flag("end-time").Value.String(), err))
				}

				bucket, prefix := sc.splitKeyValue(args[0], "/")
				if cmd.Flag("all").Changed {
					return sc.errorHandler(sc.listAllObjectsV2(bucket, prefix, delimiter, index, fetchOwner, stime, etime))
				}
				maxKeys, err := cmd.Flags().GetInt64("maxkeys")
				if err != nil {
					maxKeys = 1000
				}
				marker := cmd.Flag("marker").Value.String()
				return sc.errorHandler(sc.listObjectsV2(bucket, prefix, delimiter, marker, maxKeys, index, fetchOwner, stime, etime))
			}

			// list all my Buckets
			return sc.errorHandler(sc.bucketList())
		},
	}
	listObjectV2Cmd.Flags().StringP("marker", "m", "", "marker")
	listObjectV2Cmd.Flags().Int64P("maxkeys", "M", 1000, "max keys")
	listObjectV2Cmd.Flags().StringP("delimiter", "d", "", "Object delimiter")
	listObjectV2Cmd.Flags().BoolP("index", "i", false, "show Object index")
	listObjectV2Cmd.Flags().BoolP("owner", "", false, "fetch owner")
	listObjectV2Cmd.Flags().BoolP("all", "", false, "list all Objects")
	listObjectV2Cmd.Flags().StringP("start-time", "", "2006-01-02T15:04:05Z", "show Objects modify-time after start-time(UTC)")
	listObjectV2Cmd.Flags().StringP("end-time", "", "2060-01-02T15:04:05Z", "show Objects modify-time before end-time(UTC)")
	rootCmd.AddCommand(listObjectV2Cmd)

	listVersionCmd := &cobra.Command{
		Use:     "list-version <bucket[/prefix]>",
		Aliases: []string{"lv"},
		Short:   "list Object versions",
		Long: `list Object versions usage:
* list Object Versions
	s3cli lv bucket-name
* list Object Versions with specified prefix
	s3cli lv bucket-name/prefix`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bucket, prefix := sc.splitKeyValue(args[0], "/")
			return sc.errorHandler(sc.listObjectVersions(bucket, prefix))
		},
	}
	rootCmd.AddCommand(listVersionCmd)

	deleteVersionCmd := &cobra.Command{
		Use:     "delete-version <bucket[/prefix]>",
		Aliases: []string{"dv"},
		Short:   "delete-version of Object",
		Long: `delete Object versions usage:
* delete a Object Version
	s3cli delete-version bucket-name/key --id version-id
* delete all Objects Versions with specified prefix
	s3cli delete-version bucket-name/prefix`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bucket, prefix := sc.splitKeyValue(args[0], "/")
			version := cmd.Flag("id").Value.String()
			return sc.errorHandler(sc.deleteObjectVersion(bucket, prefix, version))
		},
	}
	deleteVersionCmd.Flags().StringP("id", "", "", "Object versionID to delete")
	rootCmd.AddCommand(deleteVersionCmd)

	restoreObjectCmd := &cobra.Command{
		Use:     "restore <bucket/key> [versionID]",
		Aliases: []string{"restore"},
		Short:   "restore Object",
		Long: `restore Object usage:
* restore a Object
	s3cli restore bucket-name/key
* restore a Object version
	s3cli restore bucket-name/key versionID
`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			bucket, key := sc.splitKeyValue(args[0], "/")
			version := ""
			if len(args) > 1 {
				version = args[1]
			}
			err := sc.restoreObject(bucket, key, version)
			return sc.errorHandler(err)
		},
	}
	rootCmd.AddCommand(restoreObjectCmd)

	downloadObjectCmd := &cobra.Command{
		Use:     "download <bucket/key> [key...]",
		Aliases: []string{"get"},
		Short:   "download Object",
		Long: `download(get) Object usage:
* download a Object to ./
	s3cli download bucket-name/key
* download Objects to ./
	s3cli download bucket-name/key key2 key3
* presign(V4) a download Object URL
	s3cli download bucket-name/key --presign`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bucket, key := sc.splitKeyValue(args[0], "/")
			objRange := cmd.Flag("range").Value.String()
			version := cmd.Flag("version").Value.String()
			err := sc.getObject(bucket, key, objRange, version)
			if err != nil {
				return sc.errorHandler(err)
			}
			if len(args) > 1 {
				for _, k := range args[1:] {
					err := sc.getObject(bucket, k, "", "")
					if err != nil {
						return sc.errorHandler(err)
					}
				}
			}
			return nil
		},
	}
	downloadObjectCmd.Flags().StringP("range", "r", "", "Object range to download, 0-64 means [0, 64]")
	downloadObjectCmd.Flags().StringP("version", "", "", "Object version to download")
	downloadObjectCmd.Flags().BoolP("overwrite", "w", false, "overwrite local file if exist")
	rootCmd.AddCommand(downloadObjectCmd)

	catObjectCmd := &cobra.Command{
		Use:   "cat <bucket/key>",
		Short: "cat Object",
		Long: `cat Object contents usage:
* cat a Object
	s3cli cat bucket-name/key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			objRange := cmd.Flag("range").Value.String()
			version := cmd.Flag("version").Value.String()
			bucket, key := sc.splitKeyValue(args[0], "/")
			return sc.errorHandler(sc.catObject(bucket, key, objRange, version))
		},
	}
	catObjectCmd.Flags().StringP("range", "r", "", "Object range to cat, 0-64 means [0, 64]")
	catObjectCmd.Flags().StringP("version", "", "", "version to cat")
	rootCmd.AddCommand(catObjectCmd)

	renameObjectCmd := &cobra.Command{
		Use:     "rename <bucket/key> <bucket/key>",
		Aliases: []string{"ren", "mv"},
		Short:   "rename Object",
		Long: `rename Bucket/key to Bucket/key usage:
* specify destination key
	s3cli mv bucket-name/key1 bucket-name2/key2
* default destionation key
	s3cli mv bucket-name/key1 bucket-name2`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			bucket, key := sc.splitKeyValue(args[1], "/")
			if key == "" {
				_, key = sc.splitKeyValue(args[0], "/")
			}
			return sc.errorHandler(sc.renameObject(args[0], bucket, key))
		},
	}
	rootCmd.AddCommand(renameObjectCmd)

	copyObjectCmd := &cobra.Command{
		Use:     "copy <bucket/key> <bucket/key>",
		Aliases: []string{"cp"},
		Short:   "copy Object",
		Long: `copy Bucket/key to Bucket/key usage:
* spedify destination Bucket and Key
	s3cli copy bucket-src/key-src bucket-dst/key-dst
* spedify destination Bucket
	s3cli copy bucket-src/key-src bucket-dst/
* spedify destionation Key
	s3cli copy bucket-src/key-src key-dst`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var metadata map[string]*string
			for _, v := range objectMetadata {
				k, v := sc.splitKeyValue(v, ":")
				if k != "" && v != "" {
					if metadata == nil {
						metadata = make(map[string]*string)
					}
					metadata[k] = &v
				}
			}
			srcBucket, srcKey := sc.splitKeyValue(args[0], "/")
			dstBucket := ""
			dstKey := ""
			if !strings.Contains(args[1], "/") {
				dstBucket = srcBucket
				dstKey = args[1]
			} else {
				dstBucket, dstKey = sc.splitKeyValue(args[1], "/")
				if dstKey == "" {
					dstKey = srcKey
				}
			}

			return sc.errorHandler(sc.copyObject(args[0], dstBucket, dstKey, objectContentType, metadata))
		},
	}
	copyObjectCmd.Flags().StringArrayVar(&objectMetadata, "md", nil, "new Object user metadata(format Key:Value)")
	copyObjectCmd.Flags().StringVar(&objectContentType, "content-type", "", "new Object content-type")
	rootCmd.AddCommand(copyObjectCmd)

	deleteObjectCmd := &cobra.Command{
		Use:     "delete <bucket/key> [key...]",
		Aliases: []string{"rm"},
		Short:   "delete Bucket or Object(s)",
		Long: `delete Bucket or Object(s) usage:
* delete Bucket 
	s3cli delete bucket-name
* delete Bucket and all Objects
	s3cli delete bucket-name --force
* delete an Object
	s3cli delete bucket-name/key
* delete Objects
	s3cli delete bucket-name/key1 key2 key3 key4
* delete all Objects with same Prefix
	s3cli delete bucket-name/prefix --prefix`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prefixMode := cmd.Flag("prefix").Changed
			force := cmd.Flag("force").Changed
			bucket, key := sc.splitKeyValue(args[0], "/")
			if len(args) > 1 {
				args[0] = key
				return sc.errorHandler(sc.deleteObjects(bucket, args))
			}
			if prefixMode {
				return sc.errorHandler(sc.deletePrefix(bucket, key))
			}
			if key == "" {
				return sc.errorHandler(sc.deleteBucketAndObjects(bucket, force))
			}

			return sc.errorHandler(sc.deleteObject(bucket, key))

		},
	}
	deleteObjectCmd.Flags().BoolP("force", "", false, "delete Bucket and all Objects")
	deleteObjectCmd.Flags().BoolP("prefix", "", false, "delete all Objects start with specified prefix")
	rootCmd.AddCommand(deleteObjectCmd)

	mpuCreateCmd := &cobra.Command{
		Use:     "mpu-init <bucket/key>",
		Short:   "init(create) a MPU request",
		Aliases: []string{"mi"},
		Long: `create a mutiPartUpload request usage:
* init(create) a MPU request
	s3cli mpu-init bucket-name/key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bucket, key := sc.splitKeyValue(args[0], "/")
			return sc.errorHandler(sc.mpuCreate(bucket, key))
		},
	}
	rootCmd.AddCommand(mpuCreateCmd)

	mpuUploadCmd := &cobra.Command{
		Use:     "mpu-upload <bucket/key> <UploadId> <part-num:file>",
		Short:   "mpu-upload MPU part(s)",
		Aliases: []string{"mu"},
		Long: `upload a MPU Part usage:
* upload MPU part1
	s3cli mpu-upload bucket-name/key UploadId 1:localfile1
* upload MPU part2
	s3cli mpu-upload bucket-name/key UploadId 2:localfile2
* upload MPU part3 and part4
	s3cli mpu-upload bucket-name/key UploadId 3:localfile3 4:localfile4`,
		Args: cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			files := map[int64]string{}
			for _, v := range args[2:] {
				i, filename := sc.splitKeyValue(v, ":")
				if filename == "" {
					return sc.errorHandler(fmt.Errorf("unknown filename: %s", filename))
				}
				index, err := strconv.ParseInt(i, 10, 64)
				if err != nil {
					return sc.errorHandler(fmt.Errorf("invalid part-num: %v, error: %s", i, err))
				}
				files[index] = filename
			}

			bucket, key := sc.splitKeyValue(args[0], "/")
			return sc.errorHandler(sc.mpuUpload(bucket, key, args[1], files))
		},
	}
	rootCmd.AddCommand(mpuUploadCmd)

	mpuAbortCmd := &cobra.Command{
		Use:     "mpu-abort <bucket/key> <UploadId>",
		Short:   "abort a MPU request",
		Aliases: []string{"ma"},
		Long: `abort a mutiPartUpload request usage:
* abort a mpu request
	s3cli mpu-abort bucket-name/key UploadId`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			bucket, key := sc.splitKeyValue(args[0], "/")
			if bucket == "" {
				return sc.errorHandler(fmt.Errorf("unknown bucket <bucket/key>(%v)", args[0]))
			}
			if key == "" {
				return sc.errorHandler(fmt.Errorf("unknown key <bucket/key>(%v)", args[0]))
			}
			return sc.errorHandler(sc.mpuAbort(bucket, key, args[1]))
		},
	}
	rootCmd.AddCommand(mpuAbortCmd)

	mpuListCmd := &cobra.Command{
		Use:     "mpu-list <bucket/prefix>",
		Aliases: []string{"ml"},
		Short:   "list MPU",
		Long: `list mutiPartUploads usage:
* list MPU
	s3cli mpu-list bucket-name/prefix`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bucket, prefix := sc.splitKeyValue(args[0], "/")
			if bucket == "" {
				return sc.errorHandler(fmt.Errorf("unknown bucket <bucket/key>(%v)", args[0]))
			}
			return sc.errorHandler(sc.mpuList(bucket, prefix))
		},
	}
	rootCmd.AddCommand(mpuListCmd)

	mpuCompleteCmd := &cobra.Command{
		Use:     "mpu-complete <bucket/key> <UploadId> <part-etag> [<part-etag> ...]",
		Short:   "complete a MPU request",
		Aliases: []string{"mc"},
		Long: `complete a mutiPartUpload request usage:
* complete a MPU request
	s3cli mpu-complete bucket-name/key UploadId etag01 etag02 etag03`,
		Args: cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			bucket, key := sc.splitKeyValue(args[0], "/")
			if bucket == "" {
				return sc.errorHandler(fmt.Errorf("unknown bucket <bucket/key>(%v)", args[0]))
			}
			if key == "" {
				return sc.errorHandler(fmt.Errorf("unknown key <bucket/key>(%v)", args[0]))
			}
			etags := make([]string, len(args)-2)
			for i := range etags {
				etags[i] = args[i+2]
			}
			return sc.errorHandler(sc.mpuComplete(bucket, key, args[1], etags))
		},
	}
	rootCmd.AddCommand(mpuCompleteCmd)

	mpuCmd := &cobra.Command{
		Use:   "mpu <bucket[/key]> [file]",
		Short: "mpu Object(mpu-create, mpu-upload and mpu-complete)",
		Long: `mpu Object usage:
* mpu a file
	s3cli mpu bucket /path/to/file
* mpu a file to Bucket/Key
	s3cli mpu bucket-name/key /path/to/file
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var fd *os.File
			bucket, key := sc.splitKeyValue(args[0], "/")
			var metadata map[string]*string
			for _, v := range objectMetadata {
				k, v := sc.splitKeyValue(v, ":")
				if k != "" && v != "" {
					if metadata == nil {
						metadata = make(map[string]*string)
					}
					metadata[k] = &v
				}
			}
			partSize, err := strconv.ParseInt(cmd.Flag("part-size").Value.String(), 10, 64)
			if err != nil {
				return fmt.Errorf("invalid part-size %s", cmd.Flag("part-size").Value.String())
			}

			fd, err = os.Open(args[1])
			if err != nil {
				return sc.errorHandler(err)
			}
			defer fd.Close()
			if objectContentType == "" {
				objectContentType = mime.TypeByExtension(filepath.Ext(args[1]))
			}
			if key == "" {
				key = filepath.Base(args[1])
			}

			err = sc.mpu(bucket, key, objectContentType, partSize<<20, fd, metadata)

			return sc.errorHandler(err)
		},
	}
	mpuCmd.Flags().StringVar(&objectContentType, "content-type", "", "Object content-type(auto detect if not specified)")
	mpuCmd.Flags().Int64("part-size", s3manager.MinUploadPartSize>>20, "MPU part-size in MB")
	mpuCmd.Flags().StringArrayVar(&objectMetadata, "md", nil, "Object user metadata(format Key:Value)")
	rootCmd.AddCommand(mpuCmd)

	//aws s3api --endpoint-url http://172.16.3.98:9020 --profile ak1 get-object-lock-configuration --bucket mybucket
	getObjectLockConfigCmd := &cobra.Command{
		Use:     "get-object-lock-configuration <bucket>",
		Aliases: []string{"golc"},
		Short:   "get-object-lock-configuration Bucket",
		Long: `get-object-lock-configuration Object usage:
* get-object-lock-configuration of a Bucket
	s3cli get-object-lock-configuration bucket
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			err = sc.getObjectLockConfig(args[0])
			return sc.errorHandler(err)
		},
	}
	rootCmd.AddCommand(getObjectLockConfigCmd)

	putObjectLockConfigCmd := &cobra.Command{
		Use:     "put-object-lock-configuration <bucket>",
		Aliases: []string{"polc"},
		Short:   "put-object-lock-configuration Bucket",
		Long: `put-object-lock-configuration Object usage:
* Enable a Bucket lock configuration
	s3cli put-object-lock-configuration bucket Enabled
* Disable a Bucket lock configuration
	s3cli put-object-lock-configuration bucket Disable
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			err = sc.putObjectLockConfig(args[0], args[1])
			return sc.errorHandler(err)
		},
	}
	rootCmd.AddCommand(putObjectLockConfigCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
