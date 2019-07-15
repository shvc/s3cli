## s3cli
#### 1. Download prebuild binary
https://github.com/vager/s3cli/releases

#### 2. Configuration credential
Edit ~/.aws/credentials
```
[default]
aws_access_key_id=AK
aws_secret_access_key=SK

[ecs]
aws_access_key_id=AK
aws_secret_access_key=SK
```

## Usage
```
./s3cli -h
s3cli client tool for S3 Bucket/Object operation
Endpoint ENV:
                S3CLI_ENDPOINT=http://host:port (only read if flag -e is not set)

Credential ENV:
                AWS_ACCESS_KEY_ID=AK     (only read if flag -p is not set)
                AWS_ACCESS_KEY=AK        (only read if AWS_ACCESS_KEY_ID is not set)
                AWS_SECRET_ACCESS_KEY=SK (only read if flag -p is not set)
                AWS_SECRET_KEY=SK        (only read if AWS_SECRET_ACCESS_KEY is not set)

Usage:
  s3cli [command]

Available Commands:
  acl          acl Bucket or Object
  createBucket create(make) Bucket
  delete       delete(remove) Object or Bucket(Bucket and Objects)
  deleteBucket delete(remove) Bucket
  download     download Object
  getacl       get Bucket/Object acl
  head         head Bucket/Object
  help         Help about any command
  list         list Buckets or Objects
  listBuckets  list Buckets
  mpu          mpu Object
  presign      presign Object
  upload       upload Object

Flags:
      --debug             print debug log
  -e, --endpoint string   S3 endpoint(http://host:port)
  -h, --help              help for s3cli
  -p, --profile string    profile in credential file
  -R, --region string     region (default "cn-north-1")
  -v, --verbose           verbose output
      --version           version for s3cli

Use "s3cli [command] --help" for more information about a command.
```

## Example
##### Create(make) Bucket
```
./s3cli -e http://192.168.55.2:9020 -p ecs cb bucket1
```

##### List Buckets
pass endpint with -e flag  
```
./s3cli -e http://192.168.55.2:9020 -p ecs lb
bucket1"
```
or pass endpoint with ENV  
```
export S3CLI_ENDPOINT=http://192.168.55.2:9020
./s3cli -p ecs lb
bucket1
```

##### Upload file
```
./s3cli -e http://192.168.55.2:9020 -p ecs cb bucket1
./s3cli -e http://192.168.55.2:9020 -p ecs up bucket1 /etc/hosts
upload /etc/hosts to bucket1/hosts success
./s3cli -e http://192.168.55.2:9020 -p ecs up bucket1 /etc/resolv.conf -k key2
upload /etc/resolv.conf to bucket1/key2 success
```

##### Download file
```
./s3cli -e http://192.168.55.2:9020 -p ecs down bucket1 hosts
download hosts to hosts
./s3cli -e http://192.168.55.2:9020 -p ecs down bucket1 key2 resolv.conf
download key2 to resolv.conf
```

##### Presign get Object
```
./s3cli -e http://192.168.55.2:9020 -p ecs psg bucket1 hosts
```

##### Presign put Object 
```
./s3cli -e http://192.168.55.2:9020 -p ecs psg bucket1 host --put
```

##### List Objects
```
./s3cli -e http://192.168.55.2:9020 -p ecs ls bucket1
host
hosts
key1
key2
key3
```

##### List Objects with specified prefix
```
./s3cli -e http://192.168.55.2:9020 -p ecs ls bucket1 -x ke
key1
key2
key3
```

##### Delete Objects with specified prefix
```
./s3cli -e http://192.168.55.2:9020 -p ecs delete bucket1 key -x
3 Objects deleted
all 3 Objects deleted
```

##### Delete Bucket and all Objects
```
./s3cli -e http://192.168.55.2:9020 -p ecs delete bucket1
2 Objects deleted
Bucket bucket1 and 2 Objects deleted
```