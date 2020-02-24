package main

import (
	"testing"
)

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
	err := s3cli.bucketList()
	if err != nil {
		t.Errorf("listBuckets failed: %s", err)
	}
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

func Test_policyBucket(t *testing.T) {
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
