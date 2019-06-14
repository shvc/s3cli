#!/bin/sh
#
version="1.0.$(git rev-list --all --count)-$(date +'%m%d%H')"

endpoint='http://s3test.myshare.io:9090'
if [ "X$1" != "X" ]
then
  endpoint=$1
fi

echo "Building s3cli-$version"
go build -ldflags "-X main.version=$version -X main.endpoint=$endpoint"
