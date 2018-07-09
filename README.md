# Louis

[![Build Status](https://travis-ci.com/KazanExpress/louis.svg?branch=master)](https://travis-ci.com/KazanExpress/louis)
[![Coverage Status](https://coveralls.io/repos/github/KazanExpress/louis/badge.svg?branch=master)](https://coveralls.io/github/KazanExpress/louis?branch=master)

Service for uploading images to any S3 compatible storage inspired from [ospry](http://ospry.io) and [cloudinary](http://cloudinary.com).

## Command line arguments

```bash
./louis --env=<default: .env | path to file with environment variables> \
        --transforms-path=<default: ensure-transforms.json | path to file containing json description of transforms> \
        --initdb=<default: true | ensure needed tables in database>
```

## Running with docker

```bash
docker run kexpress/louis
```

### Use volumes to store sqlite db

```bash
docker run -e DATA_SOURCE_NAME=/data/db.sqlite -v /my-safe/path/to/sqlite-dir:/data kexpress/louis

```

## Environment variables

```
S3_BUCKET=<name of S3 bucket>
S3_ENDPOINT=<url to S3 api server; if not set used AWS endpoint by default>
AWS_REGION=<region where s3 is stored>
AWS_ACCESS_KEY_ID=<your access key id>
AWS_SECRET_ACCESS_KEY=<your secret key>
LOUIS_PUBLIC_KEY=<key used for uploading images>
LOUIS_SECRET_KEY=<key used for claiming images>
DATA_SOURCE_NAME=<path to sqlite db>
REDIS_CONNECTION=<connection to redis, used if transformations enabled>
TRANSFORMATIONS_ENABLED=<true/false; if true then claimed images will be transformed and uploaded to S3>
```

Example:

```
S3_BUCKET=my-bucket-name
S3_ENDPOINT=https://hb.bizmrg.com
AWS_REGION=ru-msk
AWS_ACCESS_KEY_ID=super-public-key-id
AWS_SECRET_ACCESS_KEY=super-secret-key
LOUIS_PUBLIC_KEY=well-known-public-key
LOUIS_SECRET_KEY=secret-louis-key
DATA_SOURCE_NAME=mysqlite.db
REDIS_CONNECTION=redis://password@localhost:6379/
TRANSFORMATIONS_ENABLED=false
```


## Development

If you have problems with installing dependencies or building project. 
It's highly probably that they caused by [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) or [h2non/bimg](https://github.com/h2non/bimg). Please check prerequisites and installation guides for these libs.


```bash
# libvips is needed in order to install bimg
go get -v ./...

go build ./cmd/louis
./louis
```