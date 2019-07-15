#!/bin/sh
#
# Sat Jul  6 22:15:04 CST 2019
#
version="1.2.$(git rev-list HEAD --count)-$(date +'%m%d%H')"

echo "Building Linux amd64 s3cli-$version"
GOOS=linux GOARCH=amd64 go build -ldflags " -X main.version=$version"
zip -m s3cli-$version-linux-amd64.zip s3cli

echo "Building Macos amd64 s3cli-$version"
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$version"
zip -m s3cli-$version-macos-amd64.zip s3cli

echo "Building Windows amd64 s3cli-$version"
GOOS=windows GOARCH=amd64 go build -ldflags " -X main.version=$version"
zip -m s3cli-$version-win-x64.zip s3cli.exe

