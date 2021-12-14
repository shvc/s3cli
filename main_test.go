package main

import (
	"bytes"
	"log"
	mand "math/rand"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
)

var (
	s3Backend *s3mem.Backend
)

var s3cliTest = S3Cli{
	ak:     "my-ak",
	sk:     "my-sk",
	region: s3.BucketLocationConstraintCnNorth1,
	Client: nil,
}

func TestMain(m *testing.M) {
	mand.Seed(time.Now().UTC().UnixNano())
	// init fake s3
	s3Backend = s3mem.New()
	faker := gofakes3.New(s3Backend)
	ts := httptest.NewServer(faker.Server())
	defer ts.Close()
	s3cliTest.endpoint = ts.URL
	client, err := newS3Client(&s3cliTest)
	if err != nil {
		log.Fatal("newS3Client", err)
		os.Exit(1)
	}
	s3cliTest.Client = client
	if err := s3Backend.CreateBucket(testBucketName); err != nil {
		log.Fatal("backend CreateBucket error: ", err)
		os.Exit(1)
	}

	_, err = s3Backend.PutObject(testBucketName, testObjectKey, nil, bytes.NewReader(testObjectContent), int64(len(testObjectContent)))
	if err != nil {
		log.Fatal("backend PutObject error: ", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
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
