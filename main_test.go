package main

import (
	"fmt"
	"log"
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
	Client:   nil,
}

func setEnv() error {
	err := os.Setenv("AWS_ACCESS_KEY", "Q3AM3UQ867SPQQA43P2F")
	if err != nil {
		return err
	}
	return os.Setenv("AWS_SECRET_KEY", "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG")
}

func TestMain(m *testing.M) {
	client, err := newS3Client(&s3cli)
	if err != nil {
		fmt.Println("")
		log.Fatal("newS3Client", err)
	}
	s3cli.Client = client
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
