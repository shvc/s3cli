#!/bin/sh
#

version="1.0.$(git rev-list --all --count)-$(date +'%m%d%H')"

endpoint='http://s3test.myshare.io:9090'
if [ "X$1" != "X" ]
then
  endpoint=$1
fi

echo "Building Linux amd64 s3cli-$version"
GOOS=linux GOARCH=amd64 go build -ldflags " -X main.version=$version -X main.endpoint=$endpoint"
zip -m s3cli-$version-linux-amd64.zip s3cli

echo "Building Macos amd64 s3cli-$version"
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$version -X main.endpoint=$endpoint"
zip -m s3cli-$version-macos-amd64.zip s3cli

echo "Building Windows amd64 s3cli-$version"
GOOS=windows GOARCH=amd64 go build -ldflags " -X main.version=$version -X main.endpoint=$endpoint"
zip -m s3cli-$version-win-x64.zip s3cli.exe
