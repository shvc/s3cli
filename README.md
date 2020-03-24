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
s3cli -h
S3 command-line tool usage:
Endpoint EnvVar:
	S3ENDPOINT=http://host:port (only read if flag -e is not set)

Credential EnvVar:
	AWS_ACCESS_KEY_ID=AK      (only read if flag -p is not set or --ak is not set)
	AWS_ACCESS_KEY=AK         (only read if AWS_ACCESS_KEY_ID is not set)
	AWS_SECRET_ACCESS_KEY=SK  (only read if flag -p is not set or --sk is not set)
	AWS_SECRET_KEY=SK         (only read if AWS_SECRET_ACCESS_KEY is not set)

Usage:
  s3cli [command]

Available Commands:
  acl         get/set Bucket/Object ACL
  bucket      bucket sub-command
  cat         cat Object
  copy        copy Object
  delete      delete Object or Bucket
  get         get Object
  head        head Bucket/Object
  help        Help about any command
  list        list Buckets or Bucket
  listVersion list Object versions
  mpu         mpu sub-command
  presign     presign(v2) URL
  put         put Object(s)
  rename      rename Object

Flags:
      --ak string         access key
      --debug             print debug log
  -e, --endpoint string   S3 endpoint(http://host:port)
      --expire duration   presign URL expiration (default 24h0m0s)
  -h, --help              help for s3cli
      --presign           presign URL and exit
  -p, --profile string    profile in credentials file
  -R, --region string     region (default "cn-north-1")
      --sk string         secret key
  -v, --verbose           verbose output
      --version           version for s3cli

Use "s3cli [command] --help" for more information about a command.
```

## Example
#### Bucket ( s3cli bucket -h )
```sh
# bucket(b) create(c)
s3cli -e http://192.168.55.2:9020 b c bucket-name
# or pass endpoint from ENV
export S3ENDPOINT=http://192.168.55.2:9020
s3cli b c bucket-name

# list(ls) Buckets
s3cli b ls

# bucket(b) policy(p) get/set
s3cli b p bucket-name                 # get
s3cli b p bucket-name '{policy-json}' # set

# bucket(b) acl get/set
s3cli b acl bucket-name             # get
s3cli b acl bucket-name public-read # set

# bucket(b) versioning get/set
s3cli b v bucket-name

# bucket(b) delete(d)  
s3cli b d bucket-name
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
s3cli ls bucket-name -a     # list all Objects
s3cli ls bucket-name/prefix # list Objects with specified prefix
```

- delete(rm) Object(s)  
```sh
# delete Object(s)
s3cli rm bucket-name/key      # delete an Object
s3cli rm bucket-name/dir/ -x  # delete all Objects with specified prefix(dir/)
s3cli rm bucket-name --force  # delete Bucket and all Objects

# presign(V4) an DELETE Object URL
s3cli rm bucket-name/key2 --presign
```

- presign(V2) URL  
```
s3cli ps -h
```