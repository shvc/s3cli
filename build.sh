#!/bin/sh
# Sat Jul  6 22:15:04 CST 2019
#
version="1.2.$(git rev-list HEAD --count)-$(date +'%m%d%H')"

echo "Building s3cli-$version"
go build -ldflags "-X main.version=$version"
