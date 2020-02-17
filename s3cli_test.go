package main

import (
	"os"
	"testing"
)

//	# Access Key ID
//	AWS_ACCESS_KEY_ID=AKID
//	AWS_ACCESS_KEY=AKID # only read if AWS_ACCESS_KEY_ID is not set.
//
//	# Secret Access Key
//	AWS_SECRET_ACCESS_KEY=SECRET
//	AWS_SECRET_KEY=SECRET # only read if AWS_SECRET_ACCESS_KEY is not set.

var s3cli = S3Cli{
	endpoint: "https://play.min.io:9000",
	region:   "default",
}

func setup() error {
	err := os.Setenv("AWS_ACCESS_KEY", "Q3AM3UQ867SPQQA43P2F")
	if err != nil {
		return err
	}
	return os.Setenv("AWS_SECRET_KEY", "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG")
}

func Test_newS3Client(t *testing.T) {
	// TODO:
	_, err := s3cli.newS3Client()
	if err != nil {
		t.Error(err)
	}
}

func Test_splitBucketObject(t *testing.T) {
	cases := map[string][2]string{
		"":                       {"", ""},
		"/":                      {"", ""},
		"b/":                     {"b", ""},
		"bucket/object":          {"bucket", "object"},
		"b/c.ef/fff/":            {"b", "c.ef/fff/"},
		"bucket/dir/subdir/file": {"bucket", "dir/subdir/file"},
	}

	for k, v := range cases {
		bucket, object := splitBucketObject(k)
		if bucket != v[0] || object != v[1] {
			t.Errorf("expect: %s, got: %s, %s", v, bucket, object)
		}
	}
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
	err := setup()
	if err != nil {
		t.Errorf("setup failed: %s", err)
	}
	err = s3cli.listBuckets()
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
