#!/bin/sh
#
version="1.0.$(git rev-list HEAD --count)-$(date +'%m%d%H')"

endpoint='https://play.min.io:9000'
if [ "X$1" != "X" ]
then
  endpoint=$1
fi

echo "Building s3cli-$version"
go build -ldflags "-X main.version=$version -X main.endpoint=$endpoint"
