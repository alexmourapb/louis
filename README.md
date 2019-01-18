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

| Parameter                   | Description                       | Default             | Required |
|-----------------------------|-----------------------------------|---------------------|----------|
| `LOUIS_PUBLIC_KEY`  | Key used for uploading images      |      | Yes |
| `LOUIS_SECRET_KEY` | Key used for claiming images |   | Yes |
| `MAX_IMAGE_SIZE` | Maximum size of image allowed to upload in bytes | `5242880`(~5MB) | No |
| `CORS_ALLOW_ORIGIN` | Allowed origins | `*` (allows all) | No |
| `CORS_ALLOW_HEADERS` | Allowed headers | `Authorization,Content-Type,Access-Content-Allow-Origin` | No |
| `THROTTLER_QUEUE_LENGTH` | Maximum number of parallel uploads Other requests will be queued and rejected after timeout | `10` | No |
| `THROTTLER_TIMEOUT` | Queued request will be rejected after this delay with 503 status code | `15s` | No |
| `MEMORY_WATCHER_ENABLED` | if `true` then once in interval `debug.FreeOsMemory()` will be called if current RSS is more than limit | `false` | No |
| `MEMORY_WATCHER_LIMIT_BYTES` | Maximum memory amount ignored by watcher in bytes |  `1610612736` (1.5GB) | No |
| `MEMORY_WATCHER_CHECK_INTERVAL` |  | `10m` | No |
| `CLEANUP_DELAY` | Delay in minutes after which not claimed images will be deleted | `1` | No |
| `CLEANUP_POOL_CONCURRENCY` | Number of concurrent cleanup gorutines | `10` | No |
| `S3_BUCKET` | Name of S3 bucket |  | Yes |
| `S3_ENDPOINT` | By default AWS endpoint is used Should be set if another S3 compatible storage is used | AWS S3 | No |
| `AWS_REGION` | Region where S3 is stored |  | Yes |
| `AWS_ACCESS_KEY_ID` | Your S3 access key ID |  | Yes |
| `AWS_SECRET_ACCESS_KEY` | Your S3 secret key |  | Yes |
| `REDIS_URL` |  | `:6379` | No |
| `POSTGRES_ADDRESS` | PostgreSQL database address | `127.0.0.1:5432` | No |
| `POSTGRES_DATABASE` | Database name | `postgres` | No |
| `POSTGRES_USER` | | `postgres` | No |
| `POSTGRES_PASSWORD` | | `""` | No |
| `POSTGRES_SSL_MODE` | To `enable` or `disable` [SSL mode](https://www.postgresql.org/docs/9.1/libpq-ssl.html) | `disable` | No |

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