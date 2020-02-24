package main

import (
	"testing"
)

var (
	testBucketName = []string{"bucket1, bucket2, bucket3"}
)

func Test_bucketCreate(t *testing.T) {
	err := s3cliTest.bucketCreate(testBucketName)
	if err != nil {
		t.Errorf("bucketCreate failed: %s", err)
	}
}

func Test_bucketList(t *testing.T) {
	err := s3cliTest.bucketList()
	if err != nil {
		t.Errorf("listBuckets failed: %s", err)
	}
}

func Test_bucketHead(t *testing.T) {
	for _, b := range testBucketName {
		if err := s3cliTest.bucketHead(b); err != nil {
			t.Errorf("bucketHead %s failed: %s", b, err)
		}
	}
}

func Test_bucketACLGet(t *testing.T) {
	for _, b := range testBucketName {
		if err := s3cliTest.bucketACLGet(b); err != nil {
			t.Errorf("bucketACLGet %s failed: %s", b, err)
		}
	}
}

func Test_bucketACLSet(t *testing.T) {
	for _, b := range testBucketName {
		if err := s3cliTest.bucketACLSet(b, "acl"); err != nil {
			t.Errorf("bucketACLGet %s failed: %s", b, err)
		}
	}

}

func Test_bucketPolicyGet(t *testing.T) {
	for _, b := range testBucketName {
		if err := s3cliTest.bucketPolicyGet(b); err != nil {
			t.Errorf("bucketPolicyGet %s failed: %s", b, err)
		}
	}
}

func Test_bucketPolicySet(t *testing.T) {
	for _, b := range testBucketName {
		if err := s3cliTest.bucketPolicySet(b, "policy"); err != nil {
			t.Errorf("bucketPolicySet %s failed: %s", b, err)
		}
	}
}

func Test_bucketVersioningGet(t *testing.T) {
	for _, b := range testBucketName {
		if err := s3cliTest.bucketVersioningGet(b); err != nil {
			t.Errorf("bucketVersioningGet %s failed: %s", b, err)
		}
	}
}

func Test_bucketVersioningSet(t *testing.T) {
	for _, b := range testBucketName {
		if err := s3cliTest.bucketVersioningSet(b, true); err != nil {
			t.Errorf("bucketVersioningSet %s failed: %s", b, err)
		}
	}
}

func Test_bucketDelete(t *testing.T) {
	for _, b := range testBucketName {
		if err := s3cliTest.bucketDelete(b); err != nil {
			t.Errorf("bucketDelete %s failed: %s", b, err)
		}
	}
}

func Test_putObject(t *testing.T) {
	if err := s3cliTest.putObject("bucket", "key", "filename"); err != nil {
		t.Errorf("putObject failed: %s", err)
	}
}

func Test_listObjects(t *testing.T) {
	// TODO
}

func Test_getObject(t *testing.T) {
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

func Test_mpuObject(t *testing.T) {
	// TODO
}

func Test_getObjectACL(t *testing.T) {
	// TODO
}
