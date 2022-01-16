## s3cli
s3cli is a command-line tool for uploading, retrieving and managing data in AWS S3 compatible storage service.

#### Download prebuild binary
https://github.com/vager/s3cli/releases  
- Install s3cli to `/usr/local/bin/`  
```
unzip s3cli-*.zip -d /usr/local/bin/
```

#### Usage
```shell
./s3cli -h
EnvVar:
	S3_ENDPOINT=http://host:port (only read if flag --endpoint is not set)
	AWS_PROFILE=profile          (only read if flag --profile is not set)
	AWS_ACCESS_KEY_ID=ak         (only read if flag --ak and --profile not set)
	AWS_ACCESS_KEY=ak            (only read if AWS_ACCESS_KEY_ID is not set)
	AWS_SECRET_ACCESS_KEY=sk     (only read if flag --sk and --profile not set)
	AWS_SECRET_KEY=sk            (only read if AWS_SECRET_ACCESS_KEY is not set)

Usage:
  s3cli [command]

Available Commands:
  acl            get/set Bucket/Object ACL
  cat            cat Object
  completion     Generate the autocompletion script for the specified shell
  copy           copy Object
  create-bucket  create Bucket(s)
  delete         delete Object or Bucket
  delete-version delete-version of Object
  download       download Object
  head           head Bucket or Object
  help           Help about any command
  list           list Buckets or Objects
  list-v2        list Buckets or Objects(API V2)
  list-version   list Object versions
  mpu-abort      abort a MPU request
  mpu-complete   complete a MPU request
  mpu-create     create a MPU request
  mpu-list       list MPU
  mpu-upload     mpu-upload MPU part(s)
  policy         get/set Bucket Policy
  presign        presign(V2 and not escape URL path) a request
  rename         rename Object
  restore        restore Object
  upload         upload Object(s)
  version        bucket versioning

Flags:
  -a, --ak string                     S3 access key(only read if profile not set)
      --debug                         show SDK debug log
      --dial-timeout int              http dial timeout in seconds (default 5)
  -e, --endpoint string               S3 endpoint(http://host:port)
  -h, --help                          help for s3cli
      --http-keep-alive               http Keep-Alive (default true)
  -o, --output string                 output format(verbose,simple,json,line) (default "simple")
      --path-style                    use path style (default true)
      --presign                       presign Request and exit
      --presign-exp duration          presign Request expiration duration (default 24h0m0s)
  -p, --profile string                profile in credentials file
  -R, --region string                 S3 region (default "cn-north-1")
      --response-header-timeout int   http response header timeout in seconds (default 5)
  -s, --sk string                     S3 secret key(only read if profile not set)
      --v2sign                        S3 signature v2
  -v, --version                       version for s3cli

Use "s3cli [command] --help" for more information about a command.
```

## Example
#### Bucket ( s3cli bucket -h )
```shell
# create bucket
s3cli -e http://192.168.55.2:9020 create-bucket bucket-name
# or pass endpoint from ENV
export S3_ENDPOINT=http://192.168.55.2:9020
s3cli create-bucket bucket-name

# list(ls) all Buckets
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
- upload(put) Objcet(s)  
```shell
# upload file(s)
s3cli upload bucket-name/k2 /etc/hosts           # upload a file and specify Key(k2)
s3cli upload bucket-name/k2 /etc/hosts --v2sign  # upload(V2 sign) a file and specify Key(k2)
s3cli upload bucket-name /etc/hosts              # upload a file and use filename(hosts) as Key
s3cli upload bucket-name *.txt                   # upload files and use filename as Key
s3cli upload bucket-name/dir/ *.txt              # upload files and set Prefix(dir/) to all uploaded Object
s3cli put bucket-name/key3 --presign             # presign(V4) a PUT Object URL
s3cli put bucket-name/key3 --presign --v2sign    # presign(V2) a PUT Object URL
```
- download(get) Object(s)  
```shell
# download Object(s)
s3cli download bucket-name/k1                    # download Object(k1) to current dir
s3cli download bucket-name/k1                    # download(V2 sign) Object(k1) to current dir
s3cli download bucket-name/k1 k2 k3              # download Objects(key, key1 and key2) to current dir
s3cli download bucket-name/k1 --presign          # presign(V4) a GET Object URL
s3cli download bucket-name/k1 --presign --v2sign # presign(V2) a GET Object URL
```

- list(ls) Objects  
```shell
# list Objects
s3cli list bucket-name           # list(default 1000 Objects)
s3cli list bucket-name --all     # list all Objects
s3cli list bucket-name/prefix    # list Objects with specified prefix
```

- delete(rm) Object(s)  
```shell
# delete Object(s)
s3cli delete bucket-name/key                   # delete an Object
s3cli delete bucket-name/dir/ --prefix         # delete all Objects with specified prefix(dir/)
s3cli delete bucket-name --force               # delete Bucket and all Objects
s3cli delete bucket-name/k1 --presign          # presign(V4) an DELETE Object URL
s3cli delete bucket-name/k2 --presign --v2sign # presign(V2) an DELETE Object URL
```

- presign(V2) URL with raw(not escape) URL path  
```shell
# presign URL and not escape key
s3cli presign 'bucket/key(0*1).txt'
http://192.168.55.2:9000/bucket/key(0*1).txt?AWSAccessKeyId=object_user1&Expires=1588503108&Signature=93gNcprC%2BQTvlvaBxr0EizIpehM%3D
```
