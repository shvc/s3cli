package main

import (
	"bytes"
	"log"
	mrand "math/rand"
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
	accessKey: "my-ak",
	secretKey: "my-sk",
	region:    s3.BucketLocationConstraintCnNorth1,
	Client:    nil,
}

func TestMain(m *testing.M) {
	mrand.Seed(time.Now().UTC().UnixNano())
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

	_, err = s3Backend.PutObject(testBucketName, testObjectKey, nil, bytes.NewReader(testObjectContent), int64(len(testObjectContent)), nil)
	if err != nil {
		log.Fatal("backend PutObject error: ", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}
