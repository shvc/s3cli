package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand"
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
	if err := s3cliTest.putObject(testBucketName, key, "", nil, false, bytes.NewReader(nil)); err != nil {
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
	err := s3cliTest.getObject(testBucketName, testObjectKey, "", "")
	if err != nil {
		t.Errorf("getObject failed: %s", err)
		return
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

	if err := s3cliTest.deleteObject(testBucketName, key); err != nil {
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

/*
aws help

NAME
       aws -

DESCRIPTION
       The  AWS  Command  Line  Interface is a unified tool to manage your AWS
       services.

SYNOPSIS
          aws [options] <command> <subcommand> [parameters]

       Use aws command help for information on a  specific  command.  Use  aws
       help  topics  to view a list of available help topics. The synopsis for
       each command shows its parameters and their usage. Optional  parameters
       are shown in square brackets.

OPTIONS
       --debug (boolean)

       Turn on debug logging.

       --endpoint-url (string)

       Override command's default URL with the given URL.

       --no-verify-ssl (boolean)

       By  default, the AWS CLI uses SSL when communicating with AWS services.
       For each SSL connection, the AWS CLI will verify SSL certificates. This
       option overrides the default behavior of verifying SSL certificates.

       --no-paginate (boolean)

       Disable automatic pagination.

       --output (string)
*/

/*
aws s3api cmd

usage: aws [options] <command> <subcommand> [<subcommand> ...] [parameters]
To see help text, you can run:

  aws help
  aws <command> help
  aws <command> <subcommand> help

aws: error: argument operation: Invalid choice, valid choices are:

abort-multipart-upload                   | complete-multipart-upload
copy-object                              | create-bucket
create-multipart-upload                  | delete-bucket
delete-bucket-analytics-configuration    | delete-bucket-cors
delete-bucket-encryption                 | delete-bucket-intelligent-tiering-configuration
delete-bucket-inventory-configuration    | delete-bucket-lifecycle
delete-bucket-metrics-configuration      | delete-bucket-ownership-controls
delete-bucket-policy                     | delete-bucket-replication
delete-bucket-tagging                    | delete-bucket-website
delete-object                            | delete-object-tagging
delete-objects                           | delete-public-access-block
get-bucket-accelerate-configuration      | get-bucket-acl
get-bucket-analytics-configuration       | get-bucket-cors
get-bucket-encryption                    | get-bucket-intelligent-tiering-configuration
get-bucket-inventory-configuration       | get-bucket-lifecycle
get-bucket-lifecycle-configuration       | get-bucket-location
get-bucket-logging                       | get-bucket-metrics-configuration
get-bucket-notification                  | get-bucket-notification-configuration
get-bucket-ownership-controls            | get-bucket-policy
get-bucket-policy-status                 | get-bucket-replication
get-bucket-request-payment               | get-bucket-tagging
get-bucket-versioning                    | get-bucket-website
get-object                               | get-object-acl
get-object-legal-hold                    | get-object-lock-configuration
get-object-retention                     | get-object-tagging
get-object-torrent                       | get-public-access-block
head-bucket                              | head-object
list-bucket-analytics-configurations     | list-bucket-intelligent-tiering-configurations
list-bucket-inventory-configurations     | list-bucket-metrics-configurations
list-buckets                             | list-multipart-uploads
list-object-versions                     | list-objects
list-objects-v2                          | list-parts
put-bucket-accelerate-configuration      | put-bucket-acl
put-bucket-analytics-configuration       | put-bucket-cors
put-bucket-encryption                    | put-bucket-intelligent-tiering-configuration
put-bucket-inventory-configuration       | put-bucket-lifecycle
put-bucket-lifecycle-configuration       | put-bucket-logging
put-bucket-metrics-configuration         | put-bucket-notification
put-bucket-notification-configuration    | put-bucket-ownership-controls
put-bucket-policy                        | put-bucket-replication
put-bucket-request-payment               | put-bucket-tagging
put-bucket-versioning                    | put-bucket-website
put-object                               | put-object-acl
put-object-legal-hold                    | put-object-lock-configuration
put-object-retention                     | put-object-tagging
put-public-access-block                  | restore-object
select-object-content                    | upload-part
upload-part-copy                         | write-get-object-response
wait                                     | help
*/
