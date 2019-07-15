## Build
#### 1. Install golang and git
#### 2. Clone s3cli code
```
git clone https://github.com/cchare/s3cli
```
#### 3. Build
```
go get -u github.com/aws/aws-sdk-go/aws
go get -u github.com/aws/aws-sdk-go/service/s3
go get -u github.com/spf13/cobra
go build
```
#### 4. config
Edit ~/.aws/credentials
```
[default]
aws_access_key_id=YOUR_ACCESS_KEY_ID
aws_secret_access_key=YOUR_SECRET_ACCESS_KEY

[minio]
aws_access_key_id=Q3AM3UQ867SPQQA43P2F
aws_secret_access_key=zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG


```

## Usage
```
./s3cli -h
s3cli client tool for S3 Bucket/Object operation

Usage:
  s3cli [command]

Available Commands:
  acl          acl Bucket or Object
  createBucket create Bucket
  delete       delete Bucket or Object
  deleteBucket delete bucket
  download     download Object
  getacl       get Bucket/Object acl
  head         head Bucket/Object
  help         Help about any command
  list         list Buckets or Objects in Bucket
  listBucket   list Buckets
  mpu          mpu Object
  presign      presign Object
  upload       upload Object

Flags:
  -a, --accesskey string    access key
  -c, --credential string   credentail file
  -d, --debug               print debug log
  -e, --endpoint string     endpoint (default "https://play.min.io:9000")
  -h, --help                help for s3cli
  -p, --profile string      credentail profile
  -R, --region string       s3 region (default "cn-north-1")
  -s, --secretkey string    secret key
  -v, --version             print version

Use "s3cli [command] --help" for more information about a command.
```

## eg
createBubket usage  
```
./s3cli createBucket -h
create Bucket

Usage:
  s3cli createBucket <name> [flags]

Aliases:
  createBucket, cb

Flags:
  -h, --help   help for createBucket

Global Flags:
  -a, --accesskey string    access key
  -c, --credential string   credentail file
  -d, --debug               print debug log
  -e, --endpoint string     endpoint (default "https://play.min.io:9000")
  -p, --profile string      credentail profile
  -R, --region string       s3 region (default "cn-north-1")
  -s, --secretkey string    secret key
```

create Bucket  
```
 ./s3cli -p minio -R us-east-1 cb vager001
Created bucket vager001
```

upload Object  
```
./s3cli -p minio -R us-east-1 upload vager001 /etc/hosts
Uploaded Object hosts
```

list Objects  
```
./s3cli -p minio -R us-east-1 list vager001
{
  Contents: [{
      ETag: "\"9034f95a5816bf8d7370168d6c9af633\"",
      Key: "hosts",
      LastModified: 2019-07-15 10:46:14.295 +0000 UTC,
      Owner: {
        DisplayName: "",
        ID: "02d6176db174dc93cb1b899f7c6078f08654445fe8cf1b6ce98d8855f66bdbf4"
      },
      Size: 558,
      StorageClass: "STANDARD"
    }],
  Delimiter: "",
  IsTruncated: false,
  Marker: "",
  MaxKeys: 1000,
  Name: "vager001",
  Prefix: ""
}
```

delete Object  
```
./s3cli -p minio -R us-east-1 delete vager001 hosts
delete Object success
```

delete Bucket  
```
./s3cli -p minio -R us-east-1 delete vager001
bucket vager001 deleted
```
