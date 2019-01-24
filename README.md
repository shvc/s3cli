## Build
#### 1. Install golang and git
#### 2. Clone s3cli code
#### 3. Build
```
go get -u github.com/aws/aws-sdk-go/aws
go get -u github.com/aws/aws-sdk-go/service/s3
go get -u github.com/spf13/cobra
go build
```

## Usage
```
./s3cli -h
s3cli client tool for S3 Bucket/Object operation

Usage:
  s3cli [command]

Available Commands:
  createBucket create bucket
  delete       delete Object
  deleteBucket delete bucket
  download     download Object
  help         Help about any command
  list         list Object
  listBucket   list bucket
  mpu          mpu Object
  presign      presign Object
  upload       upload Object

Flags:
  -a, --accessKey string    accessKey
  -c, --credential string   credentail file
  -d, --debug               verbose output
  -e, --endpoint string     endpoint (default "http://s3test.myshare.io:9090")
  -h, --help                help for s3cli
  -p, --profile string      credentail profile
  -s, --secretKey string    secretKey
  -v, --version             output version

Use "s3cli [command] --help" for more information about a command.
```
