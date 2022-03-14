#!/bin/sh
#
set -x -e

APP='./s3cli'
BUCKET='s3cli'
KEY='testkey'
REGION='us-east-1'

export S3_ENDPOINT='https://play.min.io:9000'
export AWS_ACCESS_KEY='Q3AM3UQ867SPQQA43P2F'
export AWS_SECRET_KEY='zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG'

echo "s3cli-`date`" > ${KEY}

${APP} --region ${REGION} list
${APP} --region ${REGION} list-v2
if ${APP} --region ${REGION} head    $BUCKET | grep -q 404
then
  ${APP} --region ${REGION} create-bucket $BUCKET
else
  echo "bucket $BUCKET already exists"
fi
${APP} --region ${REGION} head    $BUCKET
${APP} --region ${REGION} acl     $BUCKET
${APP} --region ${REGION} policy  $BUCKET '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetBucketLocation","s3:ListBucket","s3:ListBucketMultipartUploads"],"Resource":["arn:aws:s3:::s3cli"]},{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject","s3:AbortMultipartUpload","s3:DeleteObject"],"Resource":["arn:aws:s3:::s3cli/*"]}]}'
${APP} --region ${REGION} policy  $BUCKET
${APP} --region ${REGION} version $BUCKET

${APP} --region ${REGION} put  $BUCKET ${KEY} --content-type applicaiton/json --md k1:v1 --md k2:v2
${APP} --region ${REGION} head $BUCKET/${KEY}
${APP} --region ${REGION} get  $BUCKET/${KEY}
${APP} --region ${REGION} copy $BUCKET/${KEY} ${KEY}.2 --md k1:v11 --md k2:v21

${APP} --region ${REGION} put  $BUCKET/dir0/${KEY} ${KEY} --content-type applicaiton/json --md k1:v1 --md k2:v2
${APP} --region ${REGION} get  $BUCKET/dir0/${KEY} 
${APP} --region ${REGION} copy $BUCKET/dir0/${KEY} $BUCKET/copy/${KEY}
${APP} --region ${REGION} head $BUCKET/copy/${KEY}

${APP} --region ${REGION} get       $BUCKET/${KEY}        --presign
${APP} --region ${REGION} put       $BUCKET/${KEY}.upload --presign
${APP} --region ${REGION} presign   $BUCKET/${KEY}
${APP} --region ${REGION} presign   $BUCKET/${KEY} -X PUT

${APP} --region ${REGION} mpu-init $BUCKET/mpu-${KEY}
${APP} --region ${REGION} mpu-list   $BUCKET/
${APP} --region ${REGION} mpu-list   $BUCKET/mpu-${KEY}
