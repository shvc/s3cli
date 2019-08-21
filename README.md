## s3cli
#### 1. Download prebuild binary
https://github.com/vager/s3cli/releases

#### 2. Configuration credential
Add you profile to ~/.aws/credentials
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
./s3cli 
s3cli client tool for S3 Bucket/Object operation
Endpoint Envvar:
	S3_ENDPOINT=http://host:port (only read if flag -e is not set)

Credential Envvar:
	AWS_ACCESS_KEY_ID=AK      (only read if flag -p is not set)
	AWS_ACCESS_KEY=AK         (only read if AWS_ACCESS_KEY_ID is not set)
	AWS_SECRET_ACCESS_KEY=SK  (only read if flag -p is not set)
	AWS_SECRET_KEY=SK         (only read if AWS_SECRET_ACCESS_KEY is not set)

Usage:
  s3cli [command]

Available Commands:
  acl          acl Bucket or Object
  cat          cat Object
  copy         copy Object
  createBucket create(make) Bucket
  delete       delete(remove) Object or Bucket(Bucket and Objects)
  deleteBucket delete a empty Bucket
  download     download Object
  getacl       get Bucket/Object ACL
  head         head Bucket/Object
  help         Help about any command
  list         list Buckets or Objects
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
##### Create Bucket
```
./s3cli -e http://192.168.55.2:9020 -p ecs cb bucket1
```

##### List Buckets
parse endpint from -e flag  
```
./s3cli -e http://192.168.55.2:9020 -p ecs ls
bucket1
```
or parse endpoint from Envvar  
```
export S3_ENDPOINT=http://192.168.55.2:9020
./s3cli -p ecs ls
bucket1
```

##### Upload file(/etc/hosts) to bucket1
```
./s3cli -p ecs up /etc/hosts bucket1
upload /etc/hosts to bucket1 success
./s3cli -p ecs up /etc/hosts bucket1/host2
upload /etc/hosts to bucket1/host2 success
```

##### Download file
```
./s3cli -p ecs down bucket1/hosts
download bucket1/hosts to hosts
s3cli down bucket1/host2 host2
download bucket1/host2 to host2
```

##### Presign URL
presign a Get URL  
```
s3cli -p ecs ps bucket1/hosts
```
presing a Put URL  
```
./s3cli -p ecs psg bucket1/hosts --put
```

##### List Objects
```
./s3cli -p ecs ls bucket1
```
List Objects with specified prefix  
```
./s3cli -p ecs ls bucket1/prefix
```

##### Delete Objects
Delete Objects with specified prefix  
```
./s3cli -p ecs delete bucket1/key -x
3 Objects deleted
all 3 Objects deleted
```
Delete Bucket and all Objects  
```
./s3cli -p ecs delete bucket1
2 Objects deleted
Bucket bucket1 and 2 Objects deleted
```