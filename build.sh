#!/bin/sh
# Sat Jul  6 22:15:04 CST 2019
#
version="1.0.$(git rev-list HEAD --count)-$(date +'%m%d%H')"

endpoint='http://s3test.myshare.io:9090'
if [ "X$1" != "X" ]
then
  endpoint=$1
fi

echo "Building s3cli-$version"
go build -ldflags "-X main.version=$version -X main.endpoint=$endpoint"