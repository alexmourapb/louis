# Louis

[![Go Report Card](https://goreportcard.com/badge/github.com/KazanExpress/louis)](https://goreportcard.com/report/github.com/KazanExpress/louis)
[![License MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://img.shields.io/badge/License-MIT-brightgreen.svg)
[![Docker Build Status](https://img.shields.io/docker/build/kexpress/louis.svg)](https://hub.docker.com/r/kexpress/louis/)

Service for transforming and uploading images to any S3 compatible storage. 

Inspired from [ospry](http://ospry.io) and [cloudinary](http://cloudinary.com).

Powered by [h2non/bimg](https://github.com/h2non/bimg).

## How it works

The upload flow lets developers make cross-browser uploads directly to configured S3 cloud storage. When an image and it's transformations successfully uploaded, **Louis** sends back metadata about the image. The metadata can be used immediately, or sent up to your own server for claiming. Unclaimed images will be deleted after some time.

![louis](https://user-images.githubusercontent.com/7482065/42679463-b07be3d6-868a-11e8-97f9-61cb67532e28.png)

See [API description](/api/docs.md) for more details on how to integrate.

## Running with docker

```bash
docker run kexpress/louis
```


### Command line arguments

```bash
./louis --env=<default: .env | path to file with environment variables> \
        --transforms-path=<default: ensure-transforms.json | path to file containing json description of transforms> \
        --initdb=<default: true | ensure needed tables in database>
```

### Configuration

`Louis` is configured using environment variables or `.env` configs (see [example.env](/example.env))

List of available configuration options:

#### LOUIS_PUBLIC_KEY

Key used for uploading images. 

*Required*.

#### LOUIS_SECRET_KEY

Key used for claiming images. 

*Required*.

#### MAX_IMAGE_SIZE

Maximum size of image allowed to upload in bytes. 

Default is `5242880`(~5MB).

*Optional*.

#### CORS_ALLOW_ORIGIN

Allowed origins. 

Default is `*` (allows all). 

*Optional*.

#### CORS_ALLOW_HEADERS

Allowed headers.

Default is `Authorization,Content-Type,Access-Content-Allow-Origin`.

*Optional*.

#### THROTTLER_QUEUE_LENGTH

Maximum number of parallel uploads. Other requests will be queued and rejected after timeout.

Default is `10`.

*Optional*.

#### THROTTLER_TIMEOUT

Queued request will be rejected after this delay with 503 status code.

Default is `15s`.

*Optional*.

#### MEMORY_WATCHER_ENABLED

if `true` then once in interval `debug.FreeOsMemory()` will be called if current RSS is more than limit.

Default is `false`.

*Optional*.

#### MEMORY_WATCHER_LIMIT_BYTES

Maximum memory amount ignored by watcher in bytes. 

Default is `1610612736` (1.5GB).

**Optinoal**.

#### MEMORY_WATCHER_CHECK_INTERVAL

Default is `10m`. 

*Optional*.


#### CLEANUP_DELAY

Delay in minutes after which not claimed images will be deleted. 

Default is `1`.

*Optional*.

#### CLEANUP_POOL_CONCURRENCY

Number of concurrent cleanup gorutines. 

Default is `10`.

*Optional*.

#### S3_BUCKET

Name of S3 bucket.

*Required*.

#### S3_ENDPOINT

By default AWS endpoint is used. Should be set if another S3 compatible storage is used.

 *Optional*.

#### AWS_REGION

Region where S3 is stored.

*Required*.

#### AWS_ACCESS_KEY_ID

Your S3 access key ID.

*Required*.

#### AWS_SECRET_ACCESS_KEY

Your S3 secret key.

*Required*.

#### REDIS_URL

Default is `:6379`.  

*Optional*.

#### POSTGRES_ADDRESS

PostgreSQL database address. Default is `127.0.0.1:5432`. 

*Optional*.

#### POSTGRES_DATABASE

Database name. Default is `postgres`.

*Optional*.

#### POSTGRES_USER

Default is `postgres`.

*Optional*.

#### POSTGRES_PASSWORD

Default is empty string.

*Optional*.

#### POSTGRES_SSL_MODE

To `enable` or `disable` [SSL mode](https://www.postgresql.org/docs/9.1/libpq-ssl.html).

Default is `disable`.

*Optional*

## Development

If you have problems with installing dependencies or building project. 
It's highly probably that they caused by [h2non/bimg](https://github.com/h2non/bimg). Please check prerequisites and installation guides for these libs.

```bash
# libvips is needed in order to install bimg
go get -v ./...

go build ./cmd/louis
./louis
```

### Databases

For development purposes(e.g. for running test) is better to use docker containers with databases:

```bash
docker run -d --rm -v $(PWD)/data/pg:/var/lib/postgresql/data -e POSTGRES_PASSWORD=1234 -e POSTGRES_USER=postgres -p 5433:5432 --name pg-app postgres
docker run -d --rm -v $(PWD)/data/rd:/data -p 6379:6379 --name rds redis
```

## Monitoring with Prometheus

Metrics are exposed in port `8001` and route `/metrics`