package main

import (
	"fmt"
	"testing"
)

var s3cli = S3Cli{}

func Test_loadS3Cfg(t *testing.T) {
	cfg, err := s3cli.loadS3Cfg()
	if err != nil {
		t.Errorf("loadS3config failed")
	}
	fmt.Println(cfg)
}
