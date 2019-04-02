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

[ecs]
aws_access_key_id=YOUR_ACCESS_KEY_ID
aws_secret_access_key=YOUR_SECRET_ACCESS_KEY
```

## Usage
```
./s3cli -h
s3cli client tool for S3 Bucket/Object operation

Usage:
  s3cli [command]

Available Commands:
  createBucket create Bucket
  delete       delete Bucket or Object
  deleteBucket delete bucket
  deleteprefix delete Objects with prefix
  download     download Object
  help         Help about any command
  list         list Buckets or Objects in Bucket
  listBucket   list Buckets
  mpu          mpu Object
  presign      presign Object
  upload       upload Object

Flags:
  -a, --accessKey string    accessKey
  -c, --credential string   credentail file
  -d, --debug               print debug log
  -e, --endpoint string     endpoint (default "http://s3test.myshare.io:9090")
  -h, --help                help for s3cli
  -p, --profile string      credentail profile
  -g, --region string       region (default "cn-north-1")
  -s, --secretKey string    secretKey
  -v, --version             print version

Use "s3cli [command] --help" for more information about a command.
```

## eg
Delete all Objects with specified prefix(API/) in a bucket(bk1)  
```
$ ./s3cli -p myecs -e http://10.10.15.98:9020 deleteprefix -h
delete all Objects with prefix

Usage:
  s3cli deleteprefix <bucket> [prefix] [flags]

Aliases:
  deleteprefix, dp

Flags:
  -h, --help   help for deleteprefix

Global Flags:
  -a, --accessKey string    accessKey
  -c, --credential string   credentail file
  -d, --debug               print debug log
  -e, --endpoint string     endpoint (default "http://s3test.myshare.io:9090")
  -p, --profile string      credentail profile
  -g, --region string       region (default "cn-north-1")
  -s, --secretKey string    secretKey


$ ./s3cli -p myecs -e http://10.10.15.98:9020 deleteprefix bk1 API/
delete 437 Objects success
```
