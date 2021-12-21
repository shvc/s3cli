## s3cli
s3cli is a command-line tool for uploading, retrieving and managing data in AWS S3 and other S3 compatible storage service.

#### Download prebuild binary
https://github.com/vager/s3cli/releases  
- Install s3cli to `/usr/local/bin/`  
```
unzip s3cli-*.zip -d /usr/local/bin/
```

#### AWS credentials configuration
Add your ak/sk to `~/.aws/credentials` or use cli flag(--ak, --sk)
```
[default]
aws_access_key_id=myAccessKey
aws_secret_access_key=mySecretKey
```

#### Usage
```sh
./s3cli help
S3 command-line tool usage:
Endpoint EnvVar:
	S3_ENDPOINT=http://host:port (only read if flag -e is not set)

Credential EnvVar:
	AWS_ACCESS_KEY_ID=AK      (only read if flag -p is not set or --ak is not set)
	AWS_ACCESS_KEY=AK         (only read if AWS_ACCESS_KEY_ID is not set)
	AWS_SECRET_ACCESS_KEY=SK  (only read if flag -p is not set or --sk is not set)
	AWS_SECRET_KEY=SK         (only read if AWS_SECRET_ACCESS_KEY is not set)

Usage:
  s3cli [command]

Available Commands:
  acl            get/set Bucket/Object ACL
  cat            cat Object
  completion     generate the autocompletion script for the specified shell
  copy           copy Object
  create-bucket  create Bucket(s)
  delete         delete Object or Bucket
  delete-version delete-version of Object
  get            get Object
  head           head Bucket or Object
  help           Help about any command
  list           list Buckets or Objects
  list-version   list Object versions
  list2          list Buckets or Objects(V2)
  mpu            mpu sub-command
  policy         get/set Bucket Policy
  presign        presign(V2) URL
  put            put Object(s)
  rename         rename Object
  restore        restore Object
  version        bucket versioning

Flags:
  -a, --ak string                     S3 access key
      --debug                         show SDK debug log
      --dial-timeout int              http dial timeout (default 5)
  -e, --endpoint string               S3 endpoint(http://host:port)
      --expire duration               presign URL expiration (default 24h0m0s)
  -h, --help                          help for s3cli
  -o, --output string                 output format(verbose,simple,json,line) (default "simple")
      --presign                       presign URL and exit
  -R, --region string                 S3 region (default "cn-north-1")
      --response-header-timeout int   http response header timeout (default 5)
  -s, --sk string                     S3 secret key
      --virtualhost                   use virtualhosting style(not use path style)

Use "s3cli [command] --help" for more information about a command.
```

## Example
#### Bucket ( s3cli bucket -h )
```sh
# create bucket
s3cli -e http://192.168.55.2:9020 cb bucket-name
# or pass endpoint from ENV
export S3_ENDPOINT=http://192.168.55.2:9020
s3cli cb bucket-name

# list(ls) Buckets
s3cli ls

# bucket policy get/set
s3cli policy bucket-name                 # get
s3cli policy bucket-name '{policy-json}' # set

# bucket acl get/set
s3cli acl bucket-name             # get
s3cli acl bucket-name public-read # set

# bucket versioning get/set
s3cli version bucket-name

# bucket delete
s3cli delete bucket-name
```

#### Object
- put(upload) Objcet(s)  
```sh
# upload file
s3cli put bucket-name /etc/hosts       # use filename as key
s3cli put bucket-name *.txt            # upload files and use filename as key
s3cli put bucket-name/dir/ *.txt       # upload files and set prefix(dir/) to all uploaded Object
s3cli put bucket-name/key2 /etc/hosts  # specify key(key2)

# presign(V4) a PUT Object URL
s3cli put bucket-name/key3 --presign

# MPU
s3cli mpu -h
```
- get(download) Object  
```sh
# download Object
s3cli get bucket-name/key            # to . and use key as filename
s3cli down bucket-name/key /tmp/file # specify local-filename

# presign(V4) a GET Object URL
s3cli get bucket-name/key --presign
```

- list(ls) Objects  
```sh
# list Objects
s3cli ls bucket-name        # list(default 1000 Objects)
s3cli ls bucket-name --all     # list all Objects
s3cli ls bucket-name/prefix # list Objects with specified prefix
```

- delete(rm) Object(s)  
```sh
# delete Object(s)
s3cli rm bucket-name/key      # delete an Object
s3cli rm bucket-name/dir/ --prefix  # delete all Objects with specified prefix(dir/)
s3cli rm bucket-name --force  # delete Bucket and all Objects

# presign(V4) an DELETE Object URL
s3cli rm bucket-name/key2 --presign
```

- presign(V2) URL  
```
# presign URL and escape key
s3cli ps 'bucket/key(0*1).txt'
http://192.168.55.2:9000/bucket/key%280%2A1%29.txt?AWSAccessKeyId=object_user1&Expires=1588503069&Signature=dVy1V1E%2FurLvzvpiF3dYhJrNMRY%3D

# presign URL and not escape key
s3cli ps --raw 'bucket/key(0*1).txt'
http://192.168.55.2:9000/bucket/key(0*1).txt?AWSAccessKeyId=object_user1&Expires=1588503108&Signature=93gNcprC%2BQTvlvaBxr0EizIpehM%3D
```
