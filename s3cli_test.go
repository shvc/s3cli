package main

import (
	"crypto/rand"
	"encoding/hex"
	mrand "math/rand"
	"testing"
)

func randomID() string {
	buf := make([]byte, 10)
	_, err := rand.Read(buf)
	if err != nil {
		for i := 0; i < 16; i++ {
			buf[i] = byte(mrand.Intn(128))
		}
	}
	return hex.EncodeToString(buf)
}

func Test_bucketCreate(t *testing.T) {
	buckets := []string{
		randomID(),
		randomID(),
	}
	err := s3cliTest.bucketCreate(buckets)
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
	bucket := randomID()
	if err := s3cliTest.bucketHead(bucket); err == nil {
		t.Errorf("expect error got success")
	}
	if err := s3cliTest.bucketCreate([]string{bucket}); err != nil {
		t.Errorf("bucketCreate failed: %s", err)
	} else {
		if err := s3cliTest.bucketHead(bucket); err != nil {
			t.Errorf("bucketHead failed: %s", err)
		}
	}
}

func Test_bucketACLGet(t *testing.T) {
	bucket := randomID()
	if err := s3cliTest.bucketACLGet(bucket); err == nil {
		t.Errorf("expect error got success")
	}
	if err := s3cliTest.bucketCreate([]string{bucket}); err != nil {
		t.Errorf("bucketCreate failed: %s", err)
	} else {
		if err := s3cliTest.bucketACLGet(bucket); err != nil {
			t.Errorf("bucketACLGet failed: %s", err)
		}
	}
}

func Test_bucketACLSet(t *testing.T) {
	bucket := randomID()
	if err := s3cliTest.bucketACLSet(bucket, "ACL"); err == nil {
		t.Errorf("expect error got success")
	}
	if err := s3cliTest.bucketCreate([]string{bucket}); err != nil {
		t.Errorf("bucketCreate failed: %s", err)
	} else {
		if err := s3cliTest.bucketACLSet(bucket, "ACL"); err != nil {
			t.Errorf("bucketACLSet failed: %s", err)
		}
	}
}

func Test_bucketPolicyGet(t *testing.T) {
	buckets := []string{
		randomID(),
		randomID(),
	}
	for _, b := range buckets {
		if err := s3cliTest.bucketPolicyGet(b); err != nil {
			t.Errorf("bucketPolicyGet %s failed: %s", b, err)
		}
	}
}

func Test_bucketPolicySet(t *testing.T) {
	buckets := []string{
		randomID(),
		randomID(),
	}
	for _, b := range buckets {
		if err := s3cliTest.bucketPolicySet(b, "policy"); err != nil {
			t.Errorf("bucketPolicySet %s failed: %s", b, err)
		}
	}
}

func Test_bucketVersioningGet(t *testing.T) {
	buckets := []string{
		randomID(),
		randomID(),
	}
	for _, b := range buckets {
		if err := s3cliTest.bucketVersioningGet(b); err != nil {
			t.Errorf("bucketVersioningGet %s failed: %s", b, err)
		}
	}
}

func Test_bucketVersioningSet(t *testing.T) {
	buckets := []string{
		randomID(),
		randomID(),
	}
	for _, b := range buckets {
		if err := s3cliTest.bucketVersioningSet(b, true); err != nil {
			t.Errorf("bucketVersioningSet %s failed: %s", b, err)
		}
	}
}

func Test_bucketDelete(t *testing.T) {
	buckets := []string{
		randomID(),
		randomID(),
	}
	for _, b := range buckets {
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

func Test_headObject(t *testing.T) {
	if err := s3cliTest.headObject("bucket", "key", false, false); err != nil {
		t.Errorf("headObject failed: %s", err)
	}
}

func Test_getObjectACL(t *testing.T) {
	if err := s3cliTest.getObjectACL("bucket", "key"); err != nil {
		t.Errorf("getObjectACL failed: %s", err)
	}
}

func Test_setObjectACL(t *testing.T) {
	if err := s3cliTest.setObjectACL("bucket", "key", "ACL"); err != nil {
		t.Errorf("setObjectACL failed: %s", err)
	}
}

func Test_listAllObjects(t *testing.T) {
	if err := s3cliTest.listAllObjects("bucket", "prefix", "dlm", true); err != nil {
		t.Errorf("listAllObjects failed: %s", err)
	}
}

func Test_listObjects(t *testing.T) {
	if err := s3cliTest.listObjects("bucket", "prefix", "dlm", "", 1000, true); err != nil {
		t.Errorf("listObjects failed: %s", err)
	}
}

func Test_listObjectVersions(t *testing.T) {
	if err := s3cliTest.listObjectVersions("bucket"); err != nil {
		t.Errorf("listObjectVersions failed: %s", err)
	}
}

func Test_getObject(t *testing.T) {
	if err := s3cliTest.getObject("bucket", "key", "", "", "filename"); err != nil {
		t.Errorf("getObject failed: %s", err)
	}
}

func Test_catObject(t *testing.T) {
	if err := s3cliTest.catObject("bucket", "key", "", ""); err != nil {
		t.Errorf("catObject failed: %s", err)
	}
}

func Test_renameObject(t *testing.T) {
	if err := s3cliTest.renameObject("source", "bucket", "key"); err != nil {
		t.Errorf("renameObject failed: %s", err)
	}
}

func Test_copyObject(t *testing.T) {
	if err := s3cliTest.copyObject("source", "bucket", "key"); err != nil {
		t.Errorf("copyObject failed: %s", err)
	}
}

func Test_deleteObjects(t *testing.T) {
	if err := s3cliTest.deleteObjects("bucket", "key"); err != nil {
		t.Errorf("deleteObjects failed: %s", err)
	}
}

func Test_deleteBucketAndObjects(t *testing.T) {
	if err := s3cliTest.deleteBucketAndObjects("bucket", true); err != nil {
		t.Errorf("deleteBucketAndObjects failed: %s", err)
	}
}

func Test_deleteObject(t *testing.T) {
	if err := s3cliTest.deleteObject("bucket", "key", ""); err != nil {
		t.Errorf("deleteObject failed: %s", err)
	}
}

func Test_mpuCreate(t *testing.T) {
	if err := s3cliTest.mpuCreate("bucket", "key"); err != nil {
		t.Errorf("mpuCreate failed: %s", err)
	}
}

func Test_mpuUpload(t *testing.T) {
	if err := s3cliTest.mpuUpload("bucket", "key", "upload-id", 1, "filename"); err != nil {
		t.Errorf("mpuUpload failed: %s", err)
	}
}

func Test_mpuAbort(t *testing.T) {
	if err := s3cliTest.mpuAbort("bucket", "key", "upload-id"); err != nil {
		t.Errorf("mpuAbort failed: %s", err)
	}
}

func Test_mpuList(t *testing.T) {
	if err := s3cliTest.mpuList("bucket", "prefix"); err != nil {
		t.Errorf("mpuList failed: %s", err)
	}
}

func Test_mpuComplete(t *testing.T) {
	if err := s3cliTest.mpuComplete("bucket", "key", "upload-id", []string{"tag1", "tag2"}); err != nil {
		t.Errorf("mpuComplete failed: %s", err)
	}
}
