## s3cli
#### 1. Download prebuild binary
https://github.com/vager/s3cli/releases

#### 2. Install s3cli to /usr/local/bin/
```
unzip s3cli-*.zip -d /usr/local/bin/
```

#### 3. Configuration credential
Add you profile(ecs) to ~/.aws/credentials
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
s3cli -h
S3 commandline tool
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
  -p, --profile string    profile in credentials file
  -R, --region string     region (default "cn-north-1")
  -v, --verbose           verbose output
      --version           version for s3cli

Use "s3cli [command] --help" for more information about a command.
```

## Example
#### Create Bucket
- parse endpoint from flag -e
```
s3cli -e http://192.168.55.2:9020 -p ecs cb bucket1
```
- or parse endpoint from Envvar  
```
export S3_ENDPOINT=http://192.168.55.2:9020
s3cli cb bucket2
```

#### Upload file
- upload file(/etc/hosts) to bucket1/hosts  
```
s3cli -p ecs up /etc/hosts bucket1
upload /etc/hosts to bucket1 success
```
- upload file(/etc/hosts) to bucket1/host2  
```
s3cli -p ecs up /etc/hosts bucket1/host2
upload /etc/hosts to bucket1/host2 success
```

#### List
- List Buckets
```
s3cli -p ecs ls
```
- List Objects(default 1000 Objects)
```
s3cli -p ecs ls bucket1
```
- List all Objects
```
s3cli -p ecs ls bucket1 -a
```
- List Objects with specified prefix
```
s3cli -p ecs ls bucket1/prefix
```

#### Download file
- download bucket1/hosts to ./hosts
```
s3cli -p ecs down bucket1/hosts
download bucket1/hosts to hosts
```
- download bucket1/hosts to /tmp/newfile
```
s3cli down bucket1/hosts /tmp/newfile
download bucket1/hosts to /tmp/newfile
```

##### Presign URL
- Presign a Get URL  
```
s3cli -p ecs ps bucket1/hosts
```
- Presign a Put URL  
```
s3cli -p ecs ps bucket1/host3 --put
```

#### Delete
- Delete an Object
```
s3cli -p ecs delete bucket1/key
```
- Delete all Objects with specified prefix
```
s3cli -p ecs delete bucket1/prefix -x
```
- Delete Bucket and all Objects  
```
s3cli -p ecs delete bucket1 --force
```