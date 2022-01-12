#!/bin/sh
#
set -e

BINARY='./s3cli'
BUCKET='bucket4s3cli'
KEY='key001'

if ${BINARY} head $BUCKET > /dev/null 2>&1
then
	${BINARY} delete $BUCKET --force
fi

# bucket command test
${BINARY} bucket create $BUCKET
${BINARY} bucket head $BUCKET
${BINARY} bucket list
${BINARY} bucket acl $BUCKET
${BINARY} bucket version $BUCKET
#${BINARY} bucket acl $BUCKET public-read
#${BINARY} bucket policy $BUCKET


#
${BINARY} head $BUCKET
${BINARY} put  $BUCKET /etc/hosts
${BINARY} head $BUCKET/hosts
${BINARY} get  $BUCKET/hosts /tmp/hosts

${BINARY} put  $BUCKET/dir0/hosts /etc/hosts
${BINARY} copy $BUCKET/dir0/hosts $BUCKET/copy/hosts
${BINARY} head $BUCKET/copy/hosts

${BINARY} put  $BUCKET/dir1/host2 /etc/hosts
${BINARY} head $BUCKET/dir1/host2
${BINARY} get  $BUCKET/dir1/host2 /tmp/hosts

${BINARY} put  $BUCKET *.go
${BINARY} head $BUCKET/main.go
${BINARY} put  $BUCKET/dir2/go/ *.go
${BINARY} head $BUCKET/dir2/go/main.go

${BINARY} get  $BUCKET/host --presign
${BINARY} put  $BUCKET/presign-put --presign
${BINARY} ps   $BUCKET/host
${BINARY} ps   $BUCKET/host -X PUT


${BINARY} mpu  create $BUCKET/mpu008
