[![Go](https://github.com/shvc/s3cli/actions/workflows/go.yml/badge.svg)](https://github.com/shvc/s3cli/actions/workflows/go.yml)
## s3cli
s3cli is a command-line tool for uploading, retrieving and managing data in AWS S3 compatible storage service.

#### Download prebuild [binary](https://github.com/shvc/s3cli/releases)  
#### Or build from source
```
git clone https://github.com/shvc/s3cli
cd s3cli
make
```

## Usage
#### Bucket 
```shell
# create bucket
s3cli -e http://192.168.55.2:9020 create-bucket bucket-name

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
s3cli upload bucket-name/k0 --data KKKK          # upload a Object(k0) with content KKKK
s3cli upload bucket-name/k1 /etc/hosts           # upload a file and specify Key(k1)
s3cli upload bucket-name/k2 /etc/hosts --v2sign  # upload(V2 sign) a file and specify Key(k2)
s3cli upload bucket-name /etc/hosts              # upload a file and use filename(hosts) as Key
s3cli upload bucket-name *.txt                   # upload files and use filename as Key
s3cli upload bucket-name/dir/ *.txt              # upload files and set Prefix(dir/) to all uploaded Object
s3cli put bucket-name/k3 --presign               # presign(V4) a PUT Object URL
s3cli put bucket-name/k4 --presign --v2sign      # presign(V2) a PUT Object URL
```
- download(get) Object(s)  
```shell
# download Object(s)
s3cli download bucket-name/k1                    # download Object(k1) to current dir
s3cli download bucket-name/k2 --v2sign           # download(V2 sign) Object(k2) to current dir
s3cli download bucket-name/k1 k2 k3              # download Objects(k1, k2 and k3) to current dir
s3cli download bucket-name/k1 --presign          # presign(V4) a GET Object URL
s3cli download bucket-name/k2 --presign --v2sign # presign(V2) a GET Object URL
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
s3cli delete bucket-name/k0                    # delete an Object(k0)
s3cli delete bucket-name/k1 k2 k3              # delete Objects(k1,k2,k3)
s3cli delete bucket-name/dir/ --prefix         # delete all Objects with specified prefix(dir/)
s3cli delete bucket-name --force               # delete Bucket and all Objects
s3cli delete bucket-name/k4 --presign          # presign(V4) an DELETE Object URL
s3cli delete bucket-name/k4 --presign --v2sign # presign(V2) an DELETE Object URL
```

- presign(V2) URL with raw(not escape) URL path  
```shell
# presign URL and not escape key
s3cli presign 'bucket/key(0*1).txt'
http://192.168.55.2:9000/bucket/key(0*1).txt?AWSAccessKeyId=object_user1&Expires=1588503108&Signature=93gNcprC%2BQTvlvaBxr0EizIpehM%3D
```
