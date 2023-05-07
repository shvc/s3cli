package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	outputVerbose = "verbose"
	outputV       = "v"
	outputSimple  = "simple"
	outputS       = "s"
	outputLine    = "line"
	outputL       = "l"
	outputJson    = "json"
	outputJ       = "j"
)

// S3Cli represent a S3Cli Client
type S3Cli struct {
	profile    string // profile in credentials file
	endpoint   string // Server endpoint(URL)
	accessKey  string // access-key
	secretKey  string // secret-key
	tokenKey   string
	region     string
	presign    bool // just presign
	presignExp time.Duration
	output     string
	header     []string // custom header(s)
	query      []string // custom query
	debug      int
	Client     *s3.S3 // manual init this field
}

func (sc *S3Cli) splitKeyValue(data, sep string) (string, string) {
	bo := strings.SplitN(data, sep, 2)
	if len(bo) == 2 {
		return bo[0], bo[1]
	}
	return data, ""
}

func (sc *S3Cli) addCustomHeader(req *http.Request) {
	for _, h := range sc.header {
		hk, hv := sc.splitKeyValue(h, ":")
		req.Header.Add(hk, hv)
	}

	q := req.URL.Query()
	for _, h := range sc.query {
		hk, hv := sc.splitKeyValue(h, "=")
		q.Add(hk, hv)
	}
	req.URL.RawQuery = q.Encode()
}

// presignV2Raw presign URL with raw(not escape) key(Object name).
func (sc *S3Cli) presignV2Raw(method, bucketKey, contentType string) (string, error) {
	if bucketKey == "" || bucketKey[0] == '/' {
		return "", fmt.Errorf("invalid bucket/key: %s", bucketKey)
	}
	secret, err := sc.Client.Config.Credentials.Get()
	if err != nil {
		return "", fmt.Errorf("access/secret key, %w", err)
	}

	u, err := url.Parse(fmt.Sprintf("%s/%s", sc.endpoint, bucketKey))
	if err != nil {
		return "", err
	}
	exp := strconv.FormatInt(time.Now().Unix()+int64(sc.presignExp.Seconds()), 10)

	q := u.Query()
	q.Set("AWSAccessKeyId", secret.AccessKeyID)
	q.Set("Expires", exp)

	contentMd5 := "" // header Content-MD5
	strToSign := fmt.Sprintf("%s\n%s\n%s\n%v\n%s", method, contentMd5, contentType, exp, u.EscapedPath())

	mac := hmac.New(sha1.New, []byte(secret.SecretAccessKey))
	mac.Write([]byte(strToSign))

	q.Set("Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// errorHandler
func (sc *S3Cli) errorHandler(err error) error {
	if sc.verboseOutput() {
		return err
	}
	if err != nil {
		fmt.Println(err)
	}
	return nil
}

func (sc *S3Cli) jsonOutput() (v bool) {
	if sc.output == outputJson || sc.output == outputJ {
		v = true
	}
	return
}

func (sc *S3Cli) verboseOutput() (v bool) {
	if sc.output == outputVerbose || sc.output == outputV {
		v = true
	}
	return
}

func (sc *S3Cli) lineOutput() (v bool) {
	if sc.output == outputLine || sc.output == outputL {
		v = true
	}
	return
}

func (sc *S3Cli) simpleOutput() (v bool) {
	if sc.output == outputSimple || sc.output == outputS {
		v = true
	}
	return
}

// bucketCreate create a Bucket
func (sc *S3Cli) bucketCreate(ctx context.Context, buckets []string) error {
	for _, b := range buckets {
		createBucketInput := &s3.CreateBucketInput{
			Bucket: aws.String(b),
			CreateBucketConfiguration: &s3.CreateBucketConfiguration{
				LocationConstraint: aws.String(sc.region),
			},
		}

		req, resp := sc.Client.CreateBucketRequest(createBucketInput)
		req.SetContext(ctx)
		if sc.presign {
			s, err := req.Presign(sc.presignExp)
			if err == nil {
				fmt.Println(s)
			}
			return err
		}

		sc.addCustomHeader(req.HTTPRequest)
		err := req.Send()
		if err != nil {
			return err
		}
		if sc.verboseOutput() {
			fmt.Println(resp)
		}
	}
	return nil
}

// bucketList list all my Buckets
func (sc *S3Cli) bucketList(ctx context.Context) error {
	req, resp := sc.Client.ListBucketsRequest(&s3.ListBucketsInput{})
	req.SetContext(ctx)
	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}
	if sc.verboseOutput() {
		fmt.Println(resp)
		return nil
	} else if sc.jsonOutput() {
		jo, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			fmt.Println(resp.String())
			return nil
		}
		fmt.Printf("%s", jo)
		return nil
	}
	for _, b := range resp.Buckets {
		if sc.lineOutput() {
			fmt.Println(
				aws.TimeValue(b.CreationDate).Format(time.RFC3339),
				aws.StringValue(resp.Owner.DisplayName),
				aws.StringValue(b.Name),
			)
		} else {
			fmt.Println(aws.StringValue(b.Name))
		}
	}
	return nil
}

// bucketHead head a Bucket
func (sc *S3Cli) bucketHead(ctx context.Context, bucket string) error {
	req, resp := sc.Client.HeadBucketRequest(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}

	if sc.lineOutput() {
		fmt.Println("ok")
	} else {
		fmt.Println(resp)
	}

	return nil
}

// bucketEncryptionGet get a Bucket bucketEncryptionGet
func (sc *S3Cli) bucketEncryptionGet(ctx context.Context, bucket string) error {
	req, resp := sc.Client.GetBucketEncryptionRequest(&s3.GetBucketEncryptionInput{
		Bucket: aws.String(bucket),
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}

	if sc.jsonOutput() {
		jo, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			fmt.Println(resp.String())
			return nil
		}
		fmt.Printf("%s", jo)
	} else {
		fmt.Println(resp.String())
	}

	return nil
}

// bucketEncryptionPut put a Bucket bucketEncryptionGet
func (sc *S3Cli) bucketEncryptionPut(ctx context.Context, bucket, algorithm string) error {
	req, resp := sc.Client.PutBucketEncryptionRequest(&s3.PutBucketEncryptionInput{
		Bucket: aws.String(bucket),
		ServerSideEncryptionConfiguration: &s3.ServerSideEncryptionConfiguration{
			Rules: []*s3.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &s3.ServerSideEncryptionByDefault{
						SSEAlgorithm: aws.String(algorithm),
					},
				},
			},
		},
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}

	if sc.jsonOutput() {
		jo, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			fmt.Println(resp.String())
			return nil
		}
		fmt.Printf("%s", jo)
	} else {
		fmt.Println(resp.String())
	}

	return nil
}

// bucketEncryptionDelete delete a Bucket bucketEncryption
func (sc *S3Cli) bucketEncryptionDelete(ctx context.Context, bucket string) error {
	req, resp := sc.Client.DeleteBucketEncryptionRequest(&s3.DeleteBucketEncryptionInput{
		Bucket: aws.String(bucket),
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}

	if sc.jsonOutput() {
		jo, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			fmt.Println(resp.String())
			return nil
		}
		fmt.Printf("%s", jo)
	} else {
		fmt.Println(resp.String())
	}

	return nil
}

// bucketACLGet get a Bucket's ACL
func (sc *S3Cli) bucketACLGet(ctx context.Context, bucket string) error {
	req, resp := sc.Client.GetBucketAclRequest(&s3.GetBucketAclInput{
		Bucket: aws.String(bucket),
	})
	req.SetContext(ctx)
	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}
	if resp != nil {
		fmt.Println(resp)
	}
	return err
}

// bucketACLSet set a Bucket's ACL
func (sc *S3Cli) bucketACLSet(ctx context.Context, bucket string, acl string) error {
	req, resp := sc.Client.PutBucketAclRequest(&s3.PutBucketAclInput{
		ACL:    aws.String(acl),
		Bucket: aws.String(bucket),
	})
	req.SetContext(ctx)
	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}
	if resp != nil {
		fmt.Println(resp)
	}
	return err
}

// bucketPolicyGet get a Bucket's Policy
func (sc *S3Cli) bucketPolicyGet(ctx context.Context, bucket string) error {
	req, resp := sc.Client.GetBucketPolicyRequest(&s3.GetBucketPolicyInput{
		Bucket: aws.String(bucket),
	})
	req.SetContext(ctx)
	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}
	fmt.Println(aws.StringValue(resp.Policy))
	return nil
}

// bucketPolicySet set a Bucket's Policy
func (sc *S3Cli) bucketPolicySet(ctx context.Context, bucket, policy string) error {
	if policy == "" {
		return errors.New("empty policy")
	}

	req, resp := sc.Client.PutBucketPolicyRequest(&s3.PutBucketPolicyInput{
		Bucket: aws.String(bucket),
		Policy: aws.String(policy),
	})
	req.SetContext(ctx)
	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}
	fmt.Println(*resp)
	return nil
}

// bucketVersioningGet get a Bucket's Versioning status
func (sc *S3Cli) bucketVersioningGet(ctx context.Context, bucket string) error {
	req, resp := sc.Client.GetBucketVersioningRequest(&s3.GetBucketVersioningInput{
		Bucket: aws.String(bucket),
	})
	req.SetContext(ctx)
	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}
	fmt.Printf("BucketVersioning: %s\n", resp)
	return nil
}

// bucketVersioningSet set a Bucket's Versioning status
func (sc *S3Cli) bucketVersioningSet(ctx context.Context, bucket string, status string) error {
	req, resp := sc.Client.PutBucketVersioningRequest(&s3.PutBucketVersioningInput{
		Bucket: aws.String(bucket),
		VersioningConfiguration: &s3.VersioningConfiguration{
			Status: aws.String(status),
		},
	})
	req.SetContext(ctx)
	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}
	fmt.Printf("BucketVersioning: %s\n", resp)
	return nil
}

// bucketDelete delete a Bucket
func (sc *S3Cli) bucketDelete(ctx context.Context, bucket string) error {
	req, _ := sc.Client.DeleteBucketRequest(&s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	return err
}

func (sc *S3Cli) getBucketCors(ctx context.Context, bucket string) error {
	req, out := sc.Client.GetBucketCorsRequest(&s3.GetBucketCorsInput{
		Bucket: aws.String(bucket),
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}
	if sc.jsonOutput() {
		jo, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			fmt.Println(out.String())
			return nil
		}
		fmt.Printf("%s", jo)
	} else {
		fmt.Println(out.String())
	}

	return err
}

func (sc *S3Cli) deleteBucketCors(ctx context.Context, bucket string) error {
	req, out := sc.Client.DeleteBucketCorsRequest(&s3.DeleteBucketCorsInput{
		Bucket: aws.String(bucket),
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}
	fmt.Println(out.String())
	return err
}

func (sc *S3Cli) putBucketCors(ctx context.Context, bucket, cfgFile string) error {
	fd, err := os.Open(cfgFile)
	if err != nil {
		return err
	}
	defer fd.Close()
	corsCfg := s3.CORSConfiguration{}
	err = json.NewDecoder(fd).Decode(&corsCfg)
	if err != nil {
		return err
	}

	req, out := sc.Client.PutBucketCorsRequest(&s3.PutBucketCorsInput{
		Bucket:            aws.String(bucket),
		CORSConfiguration: &corsCfg,
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err = req.Send()
	if err != nil {
		return err
	}
	fmt.Println(out.String())
	return err
}

// putObject upload a Object
func (sc *S3Cli) putObject(ctx context.Context, bucket, key, contentType string, metadata map[string]*string, stream bool, r io.ReadSeeker) error {
	var objContentType *string
	if contentType != "" {
		objContentType = aws.String(contentType)
	}

	putObjectInput := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: objContentType,
		Metadata:    metadata,
	}

	if stream {
		putObjectInput.ContentLength = aws.Int64(0)
	}
	if !reflect.ValueOf(r).IsNil() {
		putObjectInput.Body = r
	}
	req, resp := sc.Client.PutObjectRequest(putObjectInput)
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}

	if sc.verboseOutput() {
		fmt.Println(resp.String())
	} else if sc.lineOutput() {
		fmt.Println(
			time.Now().Format(time.RFC3339),
			"upload",
			aws.StringValue(resp.ETag),
			key,
		)
	}

	return nil
}

// headObject head a Object
func (sc *S3Cli) headObject(ctx context.Context, bucket, key string, mtime, mtimestamp bool) error {
	req, resp := sc.Client.HeadObjectRequest(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}

	if resp == nil {
		return nil
	}

	if mtime {
		fmt.Println(resp.LastModified)
	} else if mtimestamp {
		fmt.Println(resp.LastModified.Unix())
	} else if sc.lineOutput() {
		fmt.Printf("%d\t%s\n", aws.Int64Value(resp.ContentLength), resp.LastModified)
	} else if sc.jsonOutput() {
		jo, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			fmt.Println(resp.String())
			return nil
		}
		fmt.Printf("%s", jo)
		return nil
	} else {
		fmt.Println(resp)
	}

	return nil
}

// getObjectLockConfig
func (sc *S3Cli) getObjectLockConfig(ctx context.Context, bucket string) error {
	req, resp := sc.Client.GetObjectLockConfigurationRequest(&s3.GetObjectLockConfigurationInput{
		Bucket: aws.String(bucket),
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}

	if resp == nil {
		return nil
	}

	if sc.jsonOutput() {
		jo, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			fmt.Println(resp.String())
			return nil
		}
		fmt.Printf("%s", jo)
		return nil
	} else {
		fmt.Println(resp)
	}

	return nil
}

// getObjectLockConfig
func (sc *S3Cli) putObjectLockConfig(ctx context.Context, bucket, enabled string) error {
	req, resp := sc.Client.PutObjectLockConfigurationRequest(&s3.PutObjectLockConfigurationInput{
		Bucket: aws.String(bucket),
		ObjectLockConfiguration: &s3.ObjectLockConfiguration{
			ObjectLockEnabled: aws.String(enabled),
			Rule: &s3.ObjectLockRule{
				DefaultRetention: &s3.DefaultRetention{
					Days: aws.Int64(2),
					Mode: aws.String("COMPLIANCE"),
				},
			},
		},
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}

	if resp == nil {
		return nil
	}

	if sc.jsonOutput() {
		jo, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			fmt.Println(resp.String())
			return nil
		}
		fmt.Printf("%s", jo)
		return nil
	} else {
		fmt.Println(resp)
	}

	return nil
}

// getObjectACL get A Object's ACL
func (sc *S3Cli) getObjectACL(ctx context.Context, bucket, key string) error {
	req, resp := sc.Client.GetObjectAclRequest(&s3.GetObjectAclInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}
	if resp != nil {
		fmt.Println(resp)
	}
	return nil
}

// setObjectACL set A Object's ACL
func (sc *S3Cli) setObjectACL(ctx context.Context, bucket, key string, acl string) error {
	req, resp := sc.Client.PutObjectAclRequest(&s3.PutObjectAclInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		ACL:    aws.String(acl),
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}
	if resp != nil {
		fmt.Println(resp)
	}
	return nil
}

// listAllObjects list all Objects in specified bucket
func (sc *S3Cli) listAllObjects(ctx context.Context, bucket, prefix, delimiter string, index bool, startTime, endTime time.Time) error {
	var i int64
	err := sc.Client.ListObjectsPagesWithContext(ctx, &s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delimiter),
	}, func(p *s3.ListObjectsOutput, _ bool) (shouldContinue bool) {
		i++
		if sc.verboseOutput() {
			fmt.Println(p)
			return true
		} else if sc.jsonOutput() {
			jo, err := json.MarshalIndent(p, "", "  ")
			if err != nil {
				fmt.Println(p)
				return true
			}
			fmt.Printf("%s", jo)
			return true
		}
		for _, p := range p.CommonPrefixes {
			if !sc.lineOutput() {
				fmt.Println(aws.StringValue(p.Prefix))
			}
		}
		for _, obj := range p.Contents {
			if obj.LastModified.Before(startTime) {
				continue
			}
			if obj.LastModified.After(endTime) {
				continue
			}
			if sc.simpleOutput() {
				fmt.Println(
					aws.StringValue(obj.StorageClass),
					aws.TimeValue(obj.LastModified).Format(time.RFC3339),
					aws.StringValue(obj.ETag),
					aws.Int64Value(obj.Size),
					aws.StringValue(obj.Owner.DisplayName),
					aws.StringValue(obj.Key),
				)
			} else if index {
				fmt.Printf("%d\t%s\n", i, aws.StringValue(obj.Key))
				i++
			} else {
				fmt.Println(aws.StringValue(obj.Key))
			}
		}
		return true
	})

	if err != nil {
		return fmt.Errorf("list all objects failed: %w", err)
	}
	return nil
}

// listAllObjectsV2 list all Objects in specified bucket
func (sc *S3Cli) listAllObjectsV2(ctx context.Context, bucket, prefix, delimiter string, index, owner bool, startTime, endTime time.Time) error {
	var i int64
	listInput := &s3.ListObjectsV2Input{
		Bucket:     aws.String(bucket),
		FetchOwner: aws.Bool(owner),
	}
	if prefix != "" {
		listInput.SetPrefix(prefix)
	}
	if delimiter != "" {
		listInput.SetDelimiter(delimiter)
	}
	err := sc.Client.ListObjectsV2PagesWithContext(ctx, listInput, func(p *s3.ListObjectsV2Output, _ bool) (shouldContinue bool) {
		i++
		if sc.verboseOutput() {
			fmt.Println(p)
			return true
		} else if sc.jsonOutput() {
			jo, err := json.MarshalIndent(p, "", "  ")
			if err != nil {
				fmt.Println(p)
				return true
			}
			fmt.Printf("%s", jo)
			return true
		}

		for _, p := range p.CommonPrefixes {
			if !sc.lineOutput() {
				fmt.Println(aws.StringValue(p.Prefix))
			}
		}
		for _, obj := range p.Contents {
			if obj.LastModified.Before(startTime) {
				continue
			}
			if obj.LastModified.After(endTime) {
				continue
			}
			if index {
				fmt.Printf("%d\t%s\n", i, aws.StringValue(obj.Key))
				i++
			} else {
				fmt.Println(aws.StringValue(obj.Key))
			}
		}
		return true
	})

	if err != nil {
		return fmt.Errorf("list all objects failed: %w", err)
	}
	return nil
}

// listObjects (S3 listBucket)list Objects in specified bucket
func (sc *S3Cli) listObjects(ctx context.Context, bucket, prefix, delimiter, marker string, maxkeys int64, index bool, startTime, endTime time.Time) error {
	listInput := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
	}
	if prefix != "" {
		listInput.SetPrefix(prefix)
	}
	if delimiter != "" {
		listInput.SetDelimiter(delimiter)
	}
	if marker != "" {
		listInput.SetMarker(marker)
	}
	if maxkeys > 0 {
		listInput.SetMaxKeys(maxkeys)
	}
	req, resp := sc.Client.ListObjectsRequest(listInput)
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	err := req.Send()
	if err != nil {
		return fmt.Errorf("list objects failed: %w", err)
	}
	if sc.verboseOutput() {
		fmt.Println(resp)
		return nil
	} else if sc.jsonOutput() {
		jo, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			fmt.Println(resp)
			return nil
		}
		fmt.Printf("%s", jo)
		return nil
	}
	for _, p := range resp.CommonPrefixes {
		if !sc.lineOutput() {
			fmt.Println(aws.StringValue(p.Prefix))
		}
	}
	for i, obj := range resp.Contents {
		if obj.LastModified.Before(startTime) {
			continue
		}
		if obj.LastModified.After(endTime) {
			continue
		}
		if sc.lineOutput() {
			fmt.Println(
				aws.StringValue(obj.StorageClass),
				aws.TimeValue(obj.LastModified).Format(time.RFC3339),
				aws.StringValue(obj.ETag),
				aws.Int64Value(obj.Size),
				aws.StringValue(obj.Owner.DisplayName),
				aws.StringValue(obj.Key),
			)
		} else if index {
			fmt.Printf("%d\t%s\n", i, aws.StringValue(obj.Key))
		} else {
			fmt.Println(aws.StringValue(obj.Key))
		}
	}
	return nil
}

// listObjectsV2 (S3 listBucket)list Objects in specified bucket
func (sc *S3Cli) listObjectsV2(ctx context.Context, bucket, prefix, delimiter, marker string, maxkeys int64, index, owner bool, startTime, endTime time.Time) error {
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}
	if prefix != "" {
		listInput.SetPrefix(prefix)
	}
	if delimiter != "" {
		listInput.SetDelimiter(delimiter)
	}
	if marker != "" {
		listInput.SetStartAfter(marker)
	}
	if maxkeys > 0 {
		listInput.SetMaxKeys(maxkeys)
	}

	req, resp := sc.Client.ListObjectsV2Request(listInput)
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	err := req.Send()
	if err != nil {
		return fmt.Errorf("list objects failed: %w", err)
	}
	if sc.verboseOutput() {
		fmt.Println(resp)
		return nil
	} else if sc.jsonOutput() {
		jo, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			fmt.Println(resp)
			return nil
		}
		fmt.Printf("%s", jo)
		return nil
	}
	for _, p := range resp.CommonPrefixes {
		if !sc.lineOutput() {
			fmt.Println(aws.StringValue(p.Prefix))
		}
	}
	for i, obj := range resp.Contents {
		if obj.LastModified.Before(startTime) {
			continue
		}
		if obj.LastModified.After(endTime) {
			continue
		}
		if sc.lineOutput() {
			fmt.Println(
				aws.StringValue(obj.StorageClass),
				aws.TimeValue(obj.LastModified).Format(time.RFC3339),
				aws.StringValue(obj.ETag),
				aws.Int64Value(obj.Size),
				aws.StringValue(obj.Owner.DisplayName),
				aws.StringValue(obj.Key))
		} else if index {
			fmt.Printf("%d\t%s\n", i, aws.StringValue(obj.Key))
		} else {
			fmt.Println(aws.StringValue(obj.Key))
		}
	}
	return nil
}

// listObjectVersions list Objects versions in Bucket
func (sc *S3Cli) listObjectVersions(ctx context.Context, bucket, prefix string) error {
	lovi := &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	}
	if prefix != "" {
		lovi.Prefix = aws.String(prefix)
	}
	req, resp := sc.Client.ListObjectVersionsRequest(lovi)
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	err := req.Send()
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	if sc.jsonOutput() {
		jo, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			fmt.Println(resp)
			return nil
		}
		fmt.Printf("%s", jo)
	} else {
		fmt.Println(resp)
	}
	return nil
}

// getObject download a Object from bucket
func (sc *S3Cli) getObject(ctx context.Context, bucket, key, oRange, version string) error {
	var objRange *string
	if oRange != "" {
		objRange = aws.String(fmt.Sprintf("bytes=%s", oRange))
	}
	var versionID *string
	if version != "" {
		versionID = aws.String(version)
	}
	req, resp := sc.Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String(key),
		VersionId: versionID,
		Range:     objRange,
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return fmt.Errorf("get object %s failed: %w", key, err)
	}
	defer resp.Body.Close()

	// Create a file to write the S3 Object contents
	filename := filepath.Base(key)
	fd, err := os.Create(filename)
	if err != nil {
		return sc.errorHandler(err)
	}
	defer fd.Close()
	_, err = io.Copy(fd, resp.Body)
	if sc.verboseOutput() {
		fmt.Println(resp)
	} else if sc.lineOutput() {
		fmt.Println(time.Now().Format(time.RFC3339), "download", filename)
	}
	return err
}

// catObject print Object contents
func (sc *S3Cli) catObject(ctx context.Context, bucket, key, oRange, version string) error {
	var objRange *string
	if oRange != "" {
		objRange = aws.String(fmt.Sprintf("bytes=%s", oRange))
	}
	var versionID *string
	if version != "" {
		versionID = aws.String(version)
	}
	req, resp := sc.Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String(key),
		VersionId: versionID,
		Range:     objRange,
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return fmt.Errorf("get object failed: %w", err)
	}
	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}

// renameObject rename Object
func (sc *S3Cli) renameObject(ctx context.Context, source, bucket, key string) error {
	// TODO: Copy and Delete Object
	return fmt.Errorf("not impl")
}

// copyObjects copy Object to destBucket/key
func (sc *S3Cli) copyObject(ctx context.Context, source, dstBucket, dstKey, contentType string, metadata map[string]*string, mdRp bool) error {
	ci := &s3.CopyObjectInput{
		CopySource: aws.String(source),
		Bucket:     aws.String(dstBucket), // The name of the destination bucket.
		Key:        aws.String(dstKey),    // The key of the destination object.
		Metadata:   metadata,
	}
	if contentType != "" {
		ci.ContentType = aws.String(contentType)
	}

	if mdRp {
		ci.MetadataDirective = aws.String(s3.MetadataDirectiveReplace)
	} else {
		ci.MetadataDirective = aws.String(s3.MetadataDirectiveCopy)
	}

	req, resp := sc.Client.CopyObjectRequest(ci)
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return fmt.Errorf("copy object failed: %w", err)
	}
	if sc.verboseOutput() {
		fmt.Println(resp)
		return nil
	}
	return nil
}

// deletePrefix delete Objects with prefix
func (sc *S3Cli) deletePrefix(ctx context.Context, bucket, prefix string) error {
	var objNum int64
	loi := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}
	for {
		req, resp := sc.Client.ListObjectsRequest(loi)
		req.SetContext(ctx)
		err := req.Send()
		if err != nil {
			return fmt.Errorf("list object failed: %w", err)
		}
		objectNum := len(resp.Contents)
		if objectNum == 0 {
			break
		}
		if sc.verboseOutput() {
			fmt.Printf("Got %d Objects, ", objectNum)
		}
		objects := make([]*s3.ObjectIdentifier, 0, 1000)
		for _, obj := range resp.Contents {
			objects = append(objects, &s3.ObjectIdentifier{Key: obj.Key})
		}
		doi := &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &s3.Delete{
				Quiet:   aws.Bool(true),
				Objects: objects,
			},
		}
		deleteReq, _ := sc.Client.DeleteObjectsRequest(doi)
		if e := deleteReq.Send(); e != nil {
			fmt.Printf("delete Objects failed: %s", e)
		} else {
			objNum = objNum + int64(objectNum)
		}
		if sc.verboseOutput() {
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

// deleteObjects delete Objects
func (sc *S3Cli) deleteObjects(ctx context.Context, bucket string, keys []string) error {
	objects := make([]*s3.ObjectIdentifier, 0, len(keys))
	for _, v := range keys {
		if v == "" {
			continue
		}
		objects = append(objects, &s3.ObjectIdentifier{Key: aws.String(v)})
	}
	doi := &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &s3.Delete{
			Quiet:   aws.Bool(true),
			Objects: objects,
		},
	}
	req, out := sc.Client.DeleteObjectsRequest(doi)
	req.SetContext(ctx)
	err := req.Send()
	if err != nil {
		return err
	}
	if sc.verboseOutput() {
		fmt.Println(out)
	} else if sc.jsonOutput() {
		jo, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			fmt.Println(out)
			return nil
		}
		fmt.Printf("%s", jo)
	}

	return nil
}

// deleteBucketAndObjects force delete a Bucket
func (sc *S3Cli) deleteBucketAndObjects(ctx context.Context, bucket string, force bool) error {
	if force {
		if err := sc.deletePrefix(ctx, bucket, ""); err != nil {
			return err
		}
	}
	return sc.bucketDelete(ctx, bucket)
}

// deleteObjectVersion delete a Object(version)
func (sc *S3Cli) deleteObjectVersion(ctx context.Context, bucket, key, versionID string) error {
	if versionID != "" {
		req, resp := sc.Client.DeleteObjectRequest(&s3.DeleteObjectInput{
			Bucket:    aws.String(bucket),
			Key:       aws.String(key),
			VersionId: aws.String(versionID),
		})
		req.SetContext(ctx)

		if sc.presign {
			s, err := req.Presign(sc.presignExp)
			if err == nil {
				fmt.Println(s)
			}
			return err
		}

		err := req.Send()
		if err != nil {
			return err
		}
		if sc.verboseOutput() {
			fmt.Println(resp)
		}
	} else {
		req, resp := sc.Client.ListObjectVersionsRequest(&s3.ListObjectVersionsInput{
			Bucket: aws.String(bucket),
			Prefix: aws.String(key),
		})
		req.SetContext(ctx)

		err := req.Send()
		if err != nil {
			return err
		}
		if resp == nil {
			return nil
		}

		for _, v := range resp.DeleteMarkers {
			req, _ := sc.Client.DeleteObjectRequest(&s3.DeleteObjectInput{
				Bucket:    aws.String(bucket),
				Key:       v.Key,
				VersionId: v.VersionId,
			})
			err := req.Send()
			if err != nil {
				return err
			}
			if sc.verboseOutput() {
				fmt.Printf("deleteMarker %s deleted\n", aws.StringValue(v.VersionId))
			}
		}

		for _, v := range resp.Versions {
			req, _ := sc.Client.DeleteObjectRequest(&s3.DeleteObjectInput{
				Bucket:    aws.String(bucket),
				Key:       v.Key,
				VersionId: v.VersionId,
			})
			err := req.Send()
			if err != nil {
				return err
			}
			if sc.verboseOutput() {
				fmt.Printf("version %s deleted\n", aws.StringValue(v.VersionId))
			}
		}
	}
	return nil
}

// deleteObject delete a Object(version)
func (sc *S3Cli) deleteObject(ctx context.Context, bucket, key string) error {
	req, resp := sc.Client.DeleteObjectRequest(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}
	if sc.verboseOutput() {
		fmt.Println(resp)
	}
	return nil
}

// restoreObject restore a Object
func (sc *S3Cli) restoreObject(ctx context.Context, bucket, key, version string) error {
	var versionID *string
	if version != "" {
		versionID = aws.String(version)
	}
	req, resp := sc.Client.RestoreObjectRequest(&s3.RestoreObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String(key),
		VersionId: versionID,
		RestoreRequest: &s3.RestoreRequest{
			Days: aws.Int64(1),
		},
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	sc.addCustomHeader(req.HTTPRequest)
	err := req.Send()
	if err != nil {
		return err
	}
	if sc.verboseOutput() {
		fmt.Println(resp)
	}
	return nil
}

// mpuCreate create Multi-Part-Upload
func (sc *S3Cli) mpuCreate(ctx context.Context, bucket, key string) error {
	req, resp := sc.Client.CreateMultipartUploadRequest(&s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	err := req.Send()
	if err != nil {
		return err
	}

	fmt.Println(resp)
	return err
}

// mpuUpload do a Multi-Part-Upload
func (sc *S3Cli) mpuUpload(ctx context.Context, bucket, key, uid string, file map[int64]string) error {
	wg := sync.WaitGroup{}
	for i, localfile := range file {
		wg.Add(1)
		go func(num int64, filename string) {
			defer wg.Done()
			fd, err := os.Open(filename)
			if err != nil {
				fmt.Printf("%2d   error %s\n", num, err)
				return
			}
			defer fd.Close()
			req, resp := sc.Client.UploadPartRequest(&s3.UploadPartInput{
				Body:       fd,
				Bucket:     aws.String(bucket),
				Key:        aws.String(key),
				PartNumber: aws.Int64(num),
				UploadId:   aws.String(uid),
			})
			req.SetContext(ctx)

			err = req.Send()
			if err != nil {
				fmt.Printf("%2d   error %s\n", num, err)
				return
			}

			if sc.verboseOutput() {
				fmt.Println(resp)
			} else {
				fmt.Printf("%2d success %s\n", num, aws.StringValue(resp.ETag))
			}
		}(i, localfile)
	}
	wg.Wait()
	return nil
}

// mpuAbort abort Multi-Part-Upload
func (sc *S3Cli) mpuAbort(ctx context.Context, bucket, key, uid string) error {
	req, resp := sc.Client.AbortMultipartUploadRequest(&s3.AbortMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uid),
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	err := req.Send()
	if err != nil {
		return err
	}

	fmt.Println(resp)
	return err
}

// mpuList list Multi-Part-Uploads
func (sc *S3Cli) mpuList(ctx context.Context, bucket, prefix string) error {
	var keyPrefix *string
	if prefix != "" {
		keyPrefix = aws.String(prefix)
	}
	req, resp := sc.Client.ListMultipartUploadsRequest(&s3.ListMultipartUploadsInput{
		Bucket: aws.String(bucket),
		Prefix: keyPrefix,
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	err := req.Send()
	if err != nil {
		return err
	}

	if sc.jsonOutput() {
		jo, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			fmt.Println(resp)
			return nil
		}
		fmt.Printf("%s", jo)
	} else {
		fmt.Println(resp)
	}

	return err
}

// mpuComplete complete Multi-Part-Upload
func (sc *S3Cli) mpuComplete(ctx context.Context, bucket, key, uid string, etags []string) error {
	parts := make([]*s3.CompletedPart, len(etags))
	for i, v := range etags {
		parts[i] = &s3.CompletedPart{
			PartNumber: aws.Int64(int64(i + 1)),
			ETag:       aws.String(v),
		}
	}
	req, resp := sc.Client.CompleteMultipartUploadRequest(&s3.CompleteMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: parts,
		},
		UploadId: aws.String(uid),
	})
	req.SetContext(ctx)

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	err := req.Send()
	if err != nil {
		return err
	}
	fmt.Println(resp)
	return err
}

func (sc *S3Cli) mpu(ctx context.Context, bucket, key, contentType string, partSize int64, r io.Reader, metadata map[string]*string) error {
	uploader := s3manager.NewUploaderWithClient(sc.Client, func(u *s3manager.Uploader) {
		u.PartSize = partSize
	})

	mi := &s3manager.UploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		Metadata: metadata,
		Body:     r,
	}
	if contentType != "" {
		mi.ContentType = aws.String(contentType)
	}
	out, err := uploader.UploadWithContext(ctx, mi)
	if err != nil {
		return err
	}
	if out != nil {
		if sc.verboseOutput() {
			fmt.Println("location :", out.Location)
			fmt.Println("uploadID :", out.UploadID)
			fmt.Println("ETag     :", aws.StringValue(out.ETag))
			fmt.Println("versionID:", aws.StringValue(out.VersionID))
		} else if sc.jsonOutput() {
			jo, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				fmt.Println(out)
				return nil
			}
			fmt.Printf("%s", jo)
		} else {
			fmt.Printf("%s %s %s %s\n", out.Location, out.UploadID, aws.StringValue(out.ETag), aws.StringValue(out.VersionID))
		}
	}
	return nil
}
