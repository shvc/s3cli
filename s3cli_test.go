package main

import (
	"fmt"
	"testing"
)

//	# Access Key ID
//	AWS_ACCESS_KEY_ID=AKID
//	AWS_ACCESS_KEY=AKID # only read if AWS_ACCESS_KEY_ID is not set.
//
//	# Secret Access Key
//	AWS_SECRET_ACCESS_KEY=SECRET
//	AWS_SECRET_KEY=SECRET=SECRET # only read if AWS_SECRET_ACCESS_KEY is not set.

var s3cli = S3Cli{}

func Test_loadS3Cfg(t *testing.T) {
	cfg, err := s3cli.loadS3Cfg()
	if err != nil {
		t.Errorf("loadS3config failed")
	}
	fmt.Println(cfg)
}

func Test_newS3Client(t *testing.T) {
	// TODO
}

func Test_createBucket(t *testing.T) {
	// TODO
}

func Test_getBucketACL(t *testing.T) {
	// TODO
}

func Test_headBucket(t *testing.T) {
	// TODO
}

func Test_deleteBucket(t *testing.T) {
	// TODO
}

func Test_listBuckets(t *testing.T) {
	// TODO
}

func Test_aclBucket(t *testing.T) {
	// TODO
}

func Test_listAllObjects(t *testing.T) {
	// TODO
}

func Test_listObjects(t *testing.T) {
	// TODO
}

func Test_getObject(t *testing.T) {
	// TODO
}

func Test_putObject(t *testing.T) {
	// TODO
}

func Test_headObject(t *testing.T) {
	// TODO
}

func Test_deleteObjects(t *testing.T) {
	// TODO
}

func Test_deleteObject(t *testing.T) {
	// TODO
}

func Test_deleteBucketAndObjects(t *testing.T) {
	// TODO
}

func Test_aclObjects(t *testing.T) {
	// TODO
}

func Test_aclObject(t *testing.T) {
	// TODO
}

func Test_mpuObject(t *testing.T) {
	// TODO
}

func Test_presignGetObject(t *testing.T) {
	// TODO
}

func Test_presignPutObject(t *testing.T) {
	// TODO
}

func Test_getObjectACL(t *testing.T) {
	// TODO
}
