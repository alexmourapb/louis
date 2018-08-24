# Louis

[![Build Status](https://travis-ci.com/KazanExpress/louis.svg?branch=master)](https://travis-ci.com/KazanExpress/louis)
[![Coverage Status](https://coveralls.io/repos/github/KazanExpress/louis/badge.svg?branch=master)](https://coveralls.io/github/KazanExpress/louis?branch=master)

Service for uploading images to any S3 compatible storage inspired from [ospry](http://ospry.io) and [cloudinary](http://cloudinary.com).

## How it works

The upload flow lets developers make cross-browser uploads directly to configured S3 cloud storage. When an image and it's transformations successfully uploaded, Louis sends back metadata about the image. The metadata can be used immediately, or sent up to your own server for claiming. In future version, unclaimed images will be deleted.

![louis](https://user-images.githubusercontent.com/7482065/42679463-b07be3d6-868a-11e8-97f9-61cb67532e28.png)

### API docs

#### Uploading image

Request:
```
POST /upload
Headers:
    Authorization: LOUIS_PUBLIC_KEY
    Content-Type: multipart/form-data
Multipart body:
    file: image
    tags: tag1, tag2, tag3
    key: "name of image[optional/usefull on migration]"
```

Response:

```json
{
    "error": "",
    "payload": {
        "key": "bdaqolfvn27g83tpe1s0",
        "originalUrl": "https://bucketname.hb.bizmrg.com/bdaqolfvn27g83tpe1s0/original.jpg",
        "transformations": {
            "original": "https://bucketname.hb.bizmrg.com/bdaqolfvn27g83tpe1s0/original.jpg",
            "super_transform": "https://bucketname.hb.bizmrg.com/bdaqolfvn27g83tpe1s0/super_transform.jpg"
        }
    }
}
```

#### Claming image

Request:
```
POST /claim
Headers:
    Authorization: LOUIS_SECRET_KEY
    Content-Type: application/json
Body:
    {
        "keys": ["bd35b7n03vdv2aen0", "oeuaoeuhstbksu234"]
    }
```

Response:

```json
{
    "error": "",
    "payload": "ok"
}
```


#### Upload image with claim

Request:
```
POST /uploadWithClaim
Headers:
    Authorization: LOUIS_SECRET_KEY
    Content-Type: multipart/form-data
Multipart body:
    file: image
    tags: tag1, tag2, tag3
    key: "name of image[optional/usefull on migration]"
```

Response the same as in `/upload`


## Command line arguments

```
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

```env
S3_BUCKET=<name of S3 bucket>
S3_ENDPOINT=https://hb.bizmrg.com <url to S3 api server; if not set used AWS endpoint by default>
AWS_REGION=ru-msk<region where s3 is stored>
AWS_ACCESS_KEY_ID=<your S3 access key id>
AWS_SECRET_ACCESS_KEY=<your S3 secret key>
LOUIS_PUBLIC_KEY=<key used for uploading images>
LOUIS_SECRET_KEY=<key used for claiming images>
REDIS_URL=:6379
CLEANUP_DELAY=1 <delay in minutes after which not claimed images will be deleted>
CLEANUP_POOL_CONCURRENCY=10 <number of concurrent cleanup gorutines>
POSTGRES_ADDRESS=127.0.0.1:5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=1234
POSTGRES_DATABASE=postgres
POSTGRES_SSL_MODE=disable <disable/enable>
MAX_IMAGE_SIZE=5242880 <in bytes>
```


## Development

If you have problems with installing dependencies or building project. 
It's highly probably that they caused by [h2non/bimg](https://github.com/h2non/bimg). Please check prerequisites and installation guides for these libs.

```bash
# libvips is needed in order to install bimg
go get -v ./...

go build ./cmd/louis
./louis
```