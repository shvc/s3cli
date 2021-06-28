package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	mrand "math/rand"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	testBucketName    = "bucket0001"
	testObjectKey     = "key0001"
	testObjectContent = []byte("testObjectContents")
)

func randomString() string {
	buf := make([]byte, 10)
	_, err := rand.Read(buf)
	if err != nil {
		for i := 0; i < 16; i++ {
			buf[i] = byte(mrand.Intn(128))
		}
	}
	return hex.EncodeToString(buf)
}

func Test_presignV2(t *testing.T) {
	_, err := s3cliTest.presignV2(http.MethodGet, "bucket/key", "")
	if err != nil {
		t.Errorf("presignV2 failed: %s", err)
	}
}

func Test_bucketCreate(t *testing.T) {
	buckets := make([]string, 3)
	for i := range buckets {
		bucket := randomString()
		if exists, err := s3Backend.BucketExists(bucket); err != nil || exists {
			continue
		}
		buckets[i] = bucket
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
	if err := s3cliTest.bucketHead(testBucketName); err != nil {
		t.Error("bucketHead error: ", err)
	}
}

func Test_bucketACLGet(t *testing.T) {
	if err := s3cliTest.bucketACLGet(testBucketName); err != nil {
		t.Error("bucketACLGet error: ", err)
	}
}

func Test_bucketACLSet(t *testing.T) {
	t.Skip("seems gofakes3 set bucketACL has bug")
	if err := s3cliTest.bucketACLSet(testBucketName, s3.BucketCannedACLPublicReadWrite); err != nil {
		t.Error("bucketACLSet error: ", err)
	}
}

func Test_bucketPolicyGet(t *testing.T) {
	if err := s3cliTest.bucketPolicyGet(testBucketName); err != nil {
		t.Error("bucketACLGet error: ", err)
	}
}

func Test_bucketPolicySet(t *testing.T) {
	t.Skip("not read to test")
	if err := s3cliTest.bucketPolicySet(testBucketName, "{}"); err != nil {
		t.Error("bucketPolicySet error: ", err)
	}
}

func Test_bucketVersioningGet(t *testing.T) {
	if err := s3cliTest.bucketVersioningGet(testBucketName); err != nil {
		t.Error("bucketVersioningGet error: ", err)
	}
}

func Test_bucketVersioningSet(t *testing.T) {
	if err := s3cliTest.bucketVersioningSet(testBucketName, s3.BucketVersioningStatusEnabled); err != nil {
		t.Errorf("bucketVersioningSet failed: %s", err)
	}
}

func Test_bucketDelete(t *testing.T) {
	bucket := "bucketToDelete"
	if err := s3Backend.CreateBucket(bucket); err != nil {
		t.Error("backend CreateBucket error: ", err)
		return
	}
	if err := s3cliTest.bucketDelete(bucket); err != nil {
		t.Errorf("bucketDelete %s failed: %s", bucket, err)
	}
}

func Test_putObject(t *testing.T) {
	key := "testPutObject"
	if err := s3cliTest.putObject(testBucketName, key, bytes.NewReader(nil)); err != nil {
		t.Errorf("putObject failed: %s", err)
		return
	}
	_, err := s3Backend.GetObject(testBucketName, key, nil)
	if err != nil {
		t.Errorf("backend GetObject failed: %s", err)
		return
	}
}

func Test_headObject(t *testing.T) {
	if err := s3cliTest.headObject(testBucketName, testObjectKey, false, false); err != nil {
		t.Errorf("headObject failed: %s", err)
	}
}

func Test_getObjectACL(t *testing.T) {
	if err := s3cliTest.getObjectACL(testBucketName, testObjectKey); err != nil {
		t.Errorf("getObjectACL failed: %s", err)
	}
}

func Test_setObjectACL(t *testing.T) {
	if err := s3cliTest.setObjectACL(testBucketName, testObjectKey, s3.ObjectCannedACLPublicRead); err != nil {
		t.Errorf("setObjectACL failed: %s", err)
	}
}

func Test_listAllObjects(t *testing.T) {
	if err := s3cliTest.listAllObjects(testBucketName, "t", "/", true, time.Time{}, time.Time{}); err != nil {
		t.Errorf("listAllObjects failed: %s", err)
	}
}

func Test_listObjects(t *testing.T) {
	if err := s3cliTest.listObjects(testBucketName, "t", "/", "", 1000, true, time.Time{}, time.Time{}); err != nil {
		t.Errorf("listObjects failed: %s", err)
	}
}

func Test_listObjectVersions(t *testing.T) {
	if err := s3cliTest.listObjectVersions(testBucketName, ""); err != nil {
		t.Errorf("listObjectVersions failed: %s", err)
	}
}

func Test_getObject(t *testing.T) {
	r, err := s3cliTest.getObject(testBucketName, testObjectKey, "", "")
	if err != nil {
		t.Errorf("getObject failed: %s", err)
		return
	}
	defer r.Close()
	data, err := ioutil.ReadAll(r)
	if err != nil {
		t.Errorf("getObject download failed: %s", err)
		return
	}
	if !bytes.Equal(data, testObjectContent) {
		// TODO
		//t.Errorf("epect %s, got %s", testObjectContent, data)
	}
}

func Test_catObject(t *testing.T) {
	if err := s3cliTest.catObject(testBucketName, testObjectKey, "", ""); err != nil {
		t.Errorf("catObject failed: %s", err)
	}
}

func Test_renameObject(t *testing.T) {
	t.Skip("not impl")
	if err := s3cliTest.renameObject("source", testBucketName, "key"); err != nil {
		t.Errorf("renameObject failed: %s", err)
	}
}

func Test_copyObject(t *testing.T) {
	source := fmt.Sprintf("%s/%s", testBucketName, testObjectKey)
	newKey := "testCopyObjectKey"
	if err := s3cliTest.copyObject(source, testBucketName, newKey); err != nil {
		t.Errorf("copyObject failed: %s", err)
		return
	}
	if _, err := s3Backend.HeadObject(testBucketName, newKey); err != nil {
		t.Errorf("copyObject backand HeadObject failed: %s", err)
	}
}

func Test_deleteObjects(t *testing.T) {
	prefix := "testPrefix"
	if err := s3cliTest.deleteObjects(testBucketName, prefix); err != nil {
		t.Errorf("deleteObjects failed: %s", err)
	}
}

func Test_deleteBucketAndObjects(t *testing.T) {
	bucket := "bucketNameToDelete"

	if err := s3Backend.CreateBucket(bucket); err != nil {
		t.Errorf("deleteBucketAndObjects backend CreateBucket failed: %s", err)
		return
	}
	_, err := s3Backend.PutObject(testBucketName, "key01", nil, bytes.NewReader(testObjectContent), int64(len(testObjectContent)))
	if err != nil {
		t.Errorf("deleteBucketAndObjects backend PutObject failed: %s", err)
		return
	}
	_, err = s3Backend.PutObject(testBucketName, "key02", nil, bytes.NewReader(testObjectContent), int64(len(testObjectContent)))
	if err != nil {
		t.Errorf("deleteBucketAndObjects backend PutObject failed: %s", err)
		return
	}

	if err := s3cliTest.deleteBucketAndObjects(bucket, true); err != nil {
		t.Errorf("deleteBucketAndObjects failed: %s", err)
	}
}

func Test_deleteObject(t *testing.T) {
	key := "keyToTestDeleteObject"
	_, err := s3Backend.PutObject(testBucketName, key, nil, bytes.NewReader(testObjectContent), int64(len(testObjectContent)))
	if err != nil {
		t.Errorf("deleteObject backend PutObject failed: %s", err)
		return
	}

	if err := s3cliTest.deleteObject(testBucketName, key, ""); err != nil {
		t.Errorf("deleteObject failed: %s", err)
	}
}

func Test_mpuCreate(t *testing.T) {
	if err := s3cliTest.mpuCreate(testBucketName, "key"); err != nil {
		t.Errorf("mpuCreate failed: %s", err)
	}
}

func Test_mpuUpload(t *testing.T) {
	t.Skip("not ready to test")
	files := map[int64]string{
		1: "filename1",
		2: "filename2",
	}
	if err := s3cliTest.mpuUpload(testBucketName, "key", "upload-id", files); err != nil {
		t.Errorf("mpuUpload failed: %s", err)
	}
}

func Test_mpuAbort(t *testing.T) {
	t.Skip("not ready to test")
	if err := s3cliTest.mpuAbort(testBucketName, "key", "upload-id"); err != nil {
		t.Errorf("mpuAbort failed: %s", err)
	}
}

func Test_mpuList(t *testing.T) {
	t.Skip("not ready to test")
	if err := s3cliTest.mpuList(testBucketName, "prefix"); err != nil {
		t.Errorf("mpuList failed: %s", err)
	}
}

func Test_mpuComplete(t *testing.T) {
	t.Skip("not ready to test")
	if err := s3cliTest.mpuComplete(testBucketName, "key", "upload-id", []string{"tag1", "tag2"}); err != nil {
		t.Errorf("mpuComplete failed: %s", err)
	}
}

// presignV2Escaped gen a presigned URL with raw key(Object name).
func presignV2Raw(method, server, bucket, key, ak, sk, contentType string, exp int64) (string, error) {
	u, err := url.Parse(fmt.Sprintf("%s/%s/%s", server, bucket, key))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("AWSAccessKeyId", ak)
	q.Set("Expires", strconv.FormatInt(exp, 10))

	strToSign := fmt.Sprintf("%s\n%s\n%s\n%d\n%s", method, "", contentType, exp, u.EscapedPath())

	mac := hmac.New(sha1.New, []byte(sk))
	mac.Write([]byte(strToSign))

	q.Set("Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	u.RawQuery = q.Encode()

	return u.String(), nil
}
