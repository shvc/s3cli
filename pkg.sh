#!/bin/sh
#
BuildDate=$(date +'%Y/%m/%d-%H:%M:%S')

Version="1.0.$(git rev-list --all --count)"

Endpoint='http://s3test.myshare.io:9090'
if [ "X$1" != "X" ]
then
  Endpoint=$1
fi

echo "Building Linux amd64 ..."
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.buildDate=$BuildDate -X main.version=$Version -X main.endpoint=$Endpoint"
zip -m s3cli-$Version-linux-amd64.zip s3cli

echo "Building Macos amd64 ..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.buildDate=$BuildDate -X main.version=$Version -X main.endpoint=$Endpoint"
zip -m s3cli-$Version-macos-amd64.zip s3cli

echo "Building Windows amd64 ..."
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.buildDate=$BuildDate -X main.version=$Version -X main.endpoint=$Endpoint"
zip -m s3cli-$Version-win-x64.zip s3cli.exe
