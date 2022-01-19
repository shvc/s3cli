#!/bin/sh
#
set -e

BINARY='./s3cli'
BUCKET='bucket4s3cli'
KEY='key001'
REGION='us-east-1'

export S3_ENDPOINT='https://play.min.io:9000'
export AWS_ACCESS_KEY='Q3AM3UQ867SPQQA43P2F'
export AWS_SECRET_KEY='zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG'

echo "s3cli-`date`" > file

${BINARY} --region ${REGION} create-bucket $BUCKET
${BINARY} --region ${REGION} head $BUCKET
${BINARY} --region ${REGION} list
${BINARY} --region ${REGION} acl $BUCKET
${BINARY} --region ${REGION} version $BUCKET

${BINARY} --region ${REGION} head $BUCKET
${BINARY} --region ${REGION} put  $BUCKET file
${BINARY} --region ${REGION} head $BUCKET/file
${BINARY} --region ${REGION} get  $BUCKET/file

${BINARY} --region ${REGION} put  $BUCKET/dir0/file file
${BINARY} --region ${REGION} copy $BUCKET/dir0/file $BUCKET/copy/file
${BINARY} --region ${REGION} head $BUCKET/copy/file

${BINARY} --region ${REGION} put  $BUCKET/dir1/file file
${BINARY} --region ${REGION} head $BUCKET/dir1/file
${BINARY} --region ${REGION} get  $BUCKET/dir1/file 

${BINARY} --region ${REGION} put  $BUCKET/s3cli-test file
${BINARY} --region ${REGION} head $BUCKET/s3cli-test
${BINARY} --region ${REGION} put  $BUCKET/dir2/ file
${BINARY} --region ${REGION} head $BUCKET/dir2/file

${BINARY} --region ${REGION} get       $BUCKET/file --presign
${BINARY} --region ${REGION} put       $BUCKET/fileput --presign
${BINARY} --region ${REGION} presign   $BUCKET/file
${BINARY} --region ${REGION} presign   $BUCKET/file -X PUT


${BINARY} --region ${REGION} mpu-create $BUCKET/mpu008
