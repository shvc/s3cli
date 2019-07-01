package main

import "testing"

func Test_newS3Client(t *testing.T) {

	client := S3Client{
		accessKey: "ak",
		secretKey: "sk",
	}

	if _, err := client.newS3Client(); err != nil {
		t.Errorf("newS3Client failed: %s", err)
	}
}
