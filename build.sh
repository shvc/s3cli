#!/bin/sh
#
BuildDate=$(date +'%Y/%m/%d-%H:%M:%S')

Version="1.0.$(git rev-list --all --count)"

Endpoint='http://s3test.myshare.io:9090'
if [ "X$1" != "X" ]
then
  Endpoint=$1
fi

echo "Building s3cli-$Version"
go build -ldflags "-X main.BuildDate=$BuildDate -X main.Version=$Version -X main.Endpoint=$Endpoint"
