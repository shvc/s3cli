package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/s3"
)

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
	Client     *s3.S3 // manual init this field
}

// presignV2 presigne URL with escaped key(Object name).
func (sc *S3Cli) presignV2(method, bucketKey, contentType string) (string, error) {
	if bucketKey == "" || bucketKey[0] == '/' {
		return "", fmt.Errorf("invalid bucket/key: %s", bucketKey)
	}

	secret, err := sc.Client.Config.Credentials.Get()
	if err != nil {
		return "", fmt.Errorf("access/secret key, %w", err)
	}

	u, err := url.Parse(sc.endpoint)
	if err != nil {
		return "", err
	}
	exp := strconv.FormatInt(time.Now().Unix()+int64(sc.presignExp.Seconds()), 10)

	q := u.Query()
	q.Set("AWSAccessKeyId", secret.AccessKeyID)
	q.Set("Expires", exp)
	u.Path = fmt.Sprintf("/%s", bucketKey)

	contentMd5 := "" // header Content-MD5
	strToSign := fmt.Sprintf("%s\n%s\n%s\n%v\n%s", method, contentMd5, contentType, exp, u.EscapedPath())

	mac := hmac.New(sha1.New, []byte(secret.SecretAccessKey))
	mac.Write([]byte(strToSign))

	q.Set("Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// presignV2Raw presigne URL with raw key(Object name).
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

// bucketCreate create a Bucket
func (sc *S3Cli) bucketCreate(buckets []string) error {
	for _, b := range buckets {
		createBucketInput := &s3.CreateBucketInput{
			Bucket: aws.String(b),
			CreateBucketConfiguration: &s3.CreateBucketConfiguration{
				LocationConstraint: aws.String(sc.region),
			},
		}
		req, resp := sc.Client.CreateBucketRequest(createBucketInput)

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
		if sc.verbose {
			fmt.Println(resp)
		}
	}
	return nil
}

// bucketList list all my Buckets
func (sc *S3Cli) bucketList() error {
	req, resp := sc.Client.ListBucketsRequest(&s3.ListBucketsInput{})

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
	if sc.verbose {
		fmt.Println(resp)
		return nil
	}
	for _, b := range resp.Buckets {
		fmt.Println(*b.Name)
	}
	return nil
}

// bucketHead head a Bucket
func (sc *S3Cli) bucketHead(bucket string) error {
	req, resp := sc.Client.HeadBucketRequest(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})

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
	if resp != nil {
		fmt.Println(resp)
	}
	return err
}

// bucketACLGet get a Bucket's ACL
func (sc *S3Cli) bucketACLGet(bucket string) error {
	req, resp := sc.Client.GetBucketAclRequest(&s3.GetBucketAclInput{
		Bucket: aws.String(bucket),
	})

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
	if resp != nil {
		fmt.Println(resp)
	}
	return err
}

// bucketACLSet set a Bucket's ACL
func (sc *S3Cli) bucketACLSet(bucket string, acl string) error {
	req, resp := sc.Client.PutBucketAclRequest(&s3.PutBucketAclInput{
		ACL:    aws.String(acl),
		Bucket: aws.String(bucket),
	})

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
	if resp != nil {
		fmt.Println(resp)
	}
	return err
}

// bucketPolicyGet get a Bucket's Policy
func (sc *S3Cli) bucketPolicyGet(bucket string) error {
	req, resp := sc.Client.GetBucketPolicyRequest(&s3.GetBucketPolicyInput{
		Bucket: aws.String(bucket),
	})

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
	fmt.Println(*resp.Policy)
	return nil
}

// bucketPolicySet set a Bucket's Policy
func (sc *S3Cli) bucketPolicySet(bucket, policy string) error {
	if policy == "" {
		return errors.New("empty policy")
	}

	req, resp := sc.Client.PutBucketPolicyRequest(&s3.PutBucketPolicyInput{
		Bucket: aws.String(bucket),
		Policy: aws.String(policy),
	})

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
	fmt.Println(*resp)
	return nil
}

// bucketVersioningGet get a Bucket's Versioning status
func (sc *S3Cli) bucketVersioningGet(bucket string) error {
	req, resp := sc.Client.GetBucketVersioningRequest(&s3.GetBucketVersioningInput{
		Bucket: aws.String(bucket),
	})

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
	fmt.Printf("BucketVersioning: %s\n", resp)
	return nil
}

// bucketVersioningSet set a Bucket's Versioning status
func (sc *S3Cli) bucketVersioningSet(bucket string, status string) error {
	req, resp := sc.Client.PutBucketVersioningRequest(&s3.PutBucketVersioningInput{
		Bucket: aws.String(bucket),
		VersioningConfiguration: &s3.VersioningConfiguration{
			Status: aws.String(status),
		},
	})

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
	fmt.Printf("BucketVersioning: %s\n", resp)
	return nil
}

// bucketDelete delete a Bucket
func (sc *S3Cli) bucketDelete(bucket string) error {
	req, _ := sc.Client.DeleteBucketRequest(&s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	})

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	err := req.Send()
	return err
}

// putObject upload a Object
func (sc *S3Cli) putObject(bucket, key, contentType string, r io.ReadSeeker) error {
	var objContentType *string
	if contentType != "" {
		objContentType = aws.String(contentType)
	}
	putObjectInput := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: objContentType,
	}
	if !reflect.ValueOf(r).IsNil() {
		putObjectInput.Body = r
	}
	req, resp := sc.Client.PutObjectRequest(putObjectInput)

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
	if sc.verbose {
		fmt.Println(resp)
	}
	return nil
}

// headObject head a Object
func (sc *S3Cli) headObject(bucket, key string, mtime, mtimestamp bool) error {
	req, resp := sc.Client.HeadObjectRequest(&s3.HeadObjectInput{
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

	err := req.Send()
	if err != nil {
		return err
	}

	if resp == nil {
		return nil
	}
	if sc.verbose {
		fmt.Println(resp)
	} else if mtime {
		fmt.Println(resp.LastModified)
	} else if mtimestamp {
		fmt.Println(resp.LastModified.Unix())
	} else {
		fmt.Printf("%d\t%s\n", *resp.ContentLength, resp.LastModified)
	}
	return nil
}

// getObjectACL get A Object's ACL
func (sc *S3Cli) getObjectACL(bucket, key string) error {
	req, resp := sc.Client.GetObjectAclRequest(&s3.GetObjectAclInput{
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
func (sc *S3Cli) setObjectACL(bucket, key string, acl string) error {
	req, resp := sc.Client.PutObjectAclRequest(&s3.PutObjectAclInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		ACL:    aws.String(acl),
	})

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
	if resp != nil {
		fmt.Println(resp)
	}
	return nil
}

// listAllObjects list all Objects in specified bucket
func (sc *S3Cli) listAllObjects(bucket, prefix, delimiter string, index bool, startTime, endTime time.Time) error {
	var i int64
	err := sc.Client.ListObjectsPages(&s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delimiter),
	}, func(p *s3.ListObjectsOutput, last bool) (shouldContinue bool) {
		fmt.Println("Page,", i)
		i++
		if sc.verbose {
			fmt.Println(p)
			return true
		}
		for _, obj := range p.Contents {
			if obj.LastModified.Before(startTime) {
				continue
			}
			if obj.LastModified.After(endTime) {
				continue
			}
			if sc.verbose {
				fmt.Println(obj)
			} else if index {
				fmt.Printf("%d\t%s\n", i, *obj.Key)
				i++
			} else {
				fmt.Println(*obj.Key)
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
func (sc *S3Cli) listAllObjectsV2(bucket, prefix, delimiter string, index, owner bool, startTime, endTime time.Time) error {
	var i int64
	err := sc.Client.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket:     aws.String(bucket),
		Prefix:     aws.String(prefix),
		Delimiter:  aws.String(delimiter),
		FetchOwner: aws.Bool(owner),
	}, func(p *s3.ListObjectsV2Output, last bool) (shouldContinue bool) {
		fmt.Println("Page,", i)
		i++
		if sc.verbose {
			fmt.Println(p)
			return true
		}
		for _, obj := range p.Contents {
			if obj.LastModified.Before(startTime) {
				continue
			}
			if obj.LastModified.After(endTime) {
				continue
			}
			if sc.verbose {
				fmt.Println(obj)
			} else if index {
				fmt.Printf("%d\t%s\n", i, *obj.Key)
				i++
			} else {
				fmt.Println(*obj.Key)
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
func (sc *S3Cli) listObjects(bucket, prefix, delimiter, marker string, maxkeys int64, index bool, startTime, endTime time.Time) error {
	req, resp := sc.Client.ListObjectsRequest(&s3.ListObjectsInput{
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

	err := req.Send()
	if err != nil {
		return fmt.Errorf("list objects failed: %w", err)
	}
	for _, p := range resp.CommonPrefixes {
		fmt.Println(*p.Prefix)
	}
	for i, obj := range resp.Contents {
		if obj.LastModified.Before(startTime) {
			continue
		}
		if obj.LastModified.After(endTime) {
			continue
		}
		if sc.verbose {
			fmt.Println(obj)
		} else if index {
			fmt.Printf("%d\t%s\n", i, *obj.Key)
		} else {
			fmt.Println(*obj.Key)
		}
	}
	return nil
}

// listObjectsV2 (S3 listBucket)list Objects in specified bucket
func (sc *S3Cli) listObjectsV2(bucket, prefix, delimiter, marker string, maxkeys int64, index, owner bool, startTime, endTime time.Time) error {
	req, resp := sc.Client.ListObjectsV2Request(&s3.ListObjectsV2Input{
		Bucket:     aws.String(bucket),
		Prefix:     aws.String(prefix),
		StartAfter: aws.String(marker),
		Delimiter:  aws.String(delimiter),
		MaxKeys:    aws.Int64(maxkeys),
		FetchOwner: aws.Bool(owner),
	})

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
	for _, p := range resp.CommonPrefixes {
		fmt.Println(*p.Prefix)
	}
	for i, obj := range resp.Contents {
		if obj.LastModified.Before(startTime) {
			continue
		}
		if obj.LastModified.After(endTime) {
			continue
		}
		if sc.verbose {
			fmt.Println(obj)
		} else if index {
			fmt.Printf("%d\t%s\n", i, *obj.Key)
		} else {
			fmt.Println(*obj.Key)
		}
	}
	return nil
}

// listObjectVersions list Objects versions in Bucket
func (sc *S3Cli) listObjectVersions(bucket, prefix string) error {
	lovi := &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	}
	if prefix != "" {
		lovi.Prefix = aws.String(prefix)
	}
	req, resp := sc.Client.ListObjectVersionsRequest(lovi)

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

	fmt.Println(resp)
	return nil
}

// getObject download a Object from bucket
func (sc *S3Cli) getObject(bucket, key, oRange, version string) (io.ReadCloser, error) {
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

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return nil, err
	}

	err := req.Send()
	if err != nil {
		return nil, fmt.Errorf("get object failed: %w", err)
	}
	return resp.Body, nil

}

// catObject print Object contents
func (sc *S3Cli) catObject(bucket, key, oRange, version string) error {
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

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	err := req.Send()
	if err != nil {
		return fmt.Errorf("get object failed: %w", err)
	}
	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}

// renameObject rename Object
func (sc *S3Cli) renameObject(source, bucket, key string) error {
	// TODO: Copy and Delete Object
	return fmt.Errorf("not impl")
}

// copyObjects copy Object to destBucket/key
func (sc *S3Cli) copyObject(source, bucket, key string) error {
	req, resp := sc.Client.CopyObjectRequest(&s3.CopyObjectInput{
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

	err := req.Send()
	if err != nil {
		return fmt.Errorf("copy object failed: %w", err)
	}
	if sc.verbose {
		fmt.Println(resp)
		return nil
	}
	return nil
}

// deleteObjects list and delete Objects
func (sc *S3Cli) deleteObjects(bucket, prefix string) error {
	var objNum int64
	loi := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}
	for {
		req, resp := sc.Client.ListObjectsRequest(loi)
		err := req.Send()
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
		if e := deleteReq.Send(); err != nil {
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
	return sc.bucketDelete(bucket)
}

// deleteObjectVersion delete a Object(version)
func (sc *S3Cli) deleteObjectVersion(bucket, key, versionID string) error {
	if versionID != "" {
		req, resp := sc.Client.DeleteObjectRequest(&s3.DeleteObjectInput{
			Bucket:    aws.String(bucket),
			Key:       aws.String(key),
			VersionId: aws.String(versionID),
		})

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
		if sc.verbose {
			fmt.Println(resp)
		}
	} else {
		req, resp := sc.Client.ListObjectVersionsRequest(&s3.ListObjectVersionsInput{
			Bucket: aws.String(bucket),
			Prefix: aws.String(key),
		})

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
			if sc.verbose {
				fmt.Printf("deleteMarker %s deleted\n", *v.VersionId)
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
			if sc.verbose {
				fmt.Printf("version %s deleted\n", *v.VersionId)
			}
		}
	}
	return nil
}

// deleteObject delete a Object(version)
func (sc *S3Cli) deleteObject(bucket, key string) error {
	req, resp := sc.Client.DeleteObjectRequest(&s3.DeleteObjectInput{
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

	err := req.Send()
	if err != nil {
		return err
	}
	if sc.verbose {
		fmt.Println(resp)
	}
	return nil
}

// restoreObject restore a Object
func (sc *S3Cli) restoreObject(bucket, key, version string) error {
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
	if sc.verbose {
		fmt.Println(resp)
	}
	return nil
}

// mpuCreate create Multi-Part-Upload
func (sc *S3Cli) mpuCreate(bucket, key string) error {
	req, resp := sc.Client.CreateMultipartUploadRequest(&s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	err := req.Send()
	if err != nil {
		return err
	}

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	fmt.Println(resp)
	return err
}

// mpuUpload do a Multi-Part-Upload
func (sc *S3Cli) mpuUpload(bucket, key, uid string, file map[int64]string) error {
	wg := sync.WaitGroup{}
	for i, localfile := range file {
		wg.Add(1)
		go func(num int64, filename string) {
			defer wg.Done()
			fd, err := os.Open(filename)
			if err != nil {
				fmt.Printf("%2d   error: %s\n", num, err)
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
			err = req.Send()
			if err != nil {
				fmt.Printf("%2d   error: %s\n", num, err)
				return
			}
			fmt.Printf("%2d success: %s\n", num, *resp.ETag)
		}(i, localfile)
	}
	wg.Wait()
	return nil
}

// mpuAbort abort Multi-Part-Upload
func (sc *S3Cli) mpuAbort(bucket, key, uid string) error {
	req, resp := sc.Client.AbortMultipartUploadRequest(&s3.AbortMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uid),
	})
	err := req.Send()
	if err != nil {
		return err
	}

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	fmt.Println(resp)
	return err
}

// mpuList list Multi-Part-Uploads
func (sc *S3Cli) mpuList(bucket, prefix string) error {
	var keyPrefix *string
	if prefix != "" {
		keyPrefix = aws.String(prefix)
	}
	req, resp := sc.Client.ListMultipartUploadsRequest(&s3.ListMultipartUploadsInput{
		Bucket: aws.String(bucket),
		Prefix: keyPrefix,
	})
	err := req.Send()
	if err != nil {
		return err
	}

	if sc.presign {
		s, err := req.Presign(sc.presignExp)
		if err == nil {
			fmt.Println(s)
		}
		return err
	}

	fmt.Println(resp)
	return err
}

// mpuComplete completa Multi-Part-Upload
func (sc *S3Cli) mpuComplete(bucket, key, uid string, etags []string) error {
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
