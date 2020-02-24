package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"

	"github.com/aws/aws-sdk-go-v2/service/s3"
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

// mpuCreate create Multi-Part-Upload
func (sc *S3Cli) mpuCreate(bucket, key string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	req := client.CreateMultipartUploadRequest(&s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	fmt.Println(resp.CreateMultipartUploadOutput)
	return err
}

// mpuUpload do a Multi-Part-Upload
func (sc *S3Cli) mpuUpload(bucket, key, uid string, pid int64, filename string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	fd, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fd.Close()
	req := client.UploadPartRequest(&s3.UploadPartInput{
		Body:       fd,
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		PartNumber: aws.Int64(pid),
		UploadId:   aws.String(uid),
	})
	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	fmt.Println(resp.UploadPartOutput)
	return err
}

// mpuAbort abort Multi-Part-Upload
func (sc *S3Cli) mpuAbort(bucket, key, uid string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	req := client.AbortMultipartUploadRequest(&s3.AbortMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uid),
	})
	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	fmt.Println(resp.AbortMultipartUploadOutput)
	return err
}

// mpuList list Multi-Part-Uploads
func (sc *S3Cli) mpuList(bucket, prefix string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	var keyPrefix *string
	if prefix != "" {
		keyPrefix = aws.String(prefix)
	}
	req := client.ListMultipartUploadsRequest(&s3.ListMultipartUploadsInput{
		Bucket: aws.String(bucket),
		Prefix: keyPrefix,
	})
	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	fmt.Println(resp.ListMultipartUploadsOutput)
	return err
}

// mpuComplete completa Multi-Part-Upload
func (sc *S3Cli) mpuComplete(bucket, key, uid string, etags []string) error {
	client, err := sc.newS3Client()
	if err != nil {
		return fmt.Errorf("init s3 Client failed: %w", err)
	}
	parts := make([]s3.CompletedPart, len(etags))
	for i, v := range etags {
		parts[i] = s3.CompletedPart{
			PartNumber: aws.Int64(int64(i + 1)),
			ETag:       aws.String(v),
		}
	}
	req := client.CompleteMultipartUploadRequest(&s3.CompleteMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: parts,
		},
		UploadId: aws.String(uid),
	})
	resp, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	fmt.Println(resp.CompleteMultipartUploadOutput)
	return err
}
