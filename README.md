# Louis [BETA]

[![Go Report Card](https://goreportcard.com/badge/github.com/KazanExpress/louis)](https://goreportcard.com/report/github.com/KazanExpress/louis)
[![License MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://img.shields.io/badge/License-MIT-brightgreen.svg)
[![Build Status](https://drone.kznexpress.ru/api/badges/KazanExpress/louis/status.svg)](https://drone.kznexpress.ru/KazanExpress/louis)

Service for transforming and uploading images to any S3 compatible storage.

Inspired from [ospry](http://ospry.io) and [cloudinary](http://cloudinary.com).

Powered by [h2non/bimg](https://github.com/h2non/bimg).

## How it works

The upload flow lets developers make cross-browser uploads directly to configured S3 cloud storage. When an image and it's transformations successfully uploaded, **Louis** sends back metadata about the image. The metadata can be used immediately, or sent up to your own server for claiming. Unclaimed images will be deleted after some time.

![louis](https://user-images.githubusercontent.com/7482065/42679463-b07be3d6-868a-11e8-97f9-61cb67532e28.png)

See [API description](/api/docs.md) for more details on how to integrate.

## Transformations

There are some implemented transformations which can be applied during the image upload.

To use transforms, there should be `ensure-transform.json` (take a look at [example file](https://github.com/KazanExpress/louis/blob/master/cmd/louis/ensure-transforms.json)) file passed to `Louis`. For now `Louis`, inserts all new transformations from file to postgres during application start. This way of configuring will be definitely changed in the future.

Each element in `transformations` of `ensure-transforms.json` describes transformation rule:

- `type` - type of transform. For now it's either `fill` or `fit`

- `name` field represents unique name of transformation, it will be used in uploaded image url for that transform

- transformation will be applied to all new images which have same `tag`

- `width` and `height` parameters for transformation (note that it's not necessarily final size of transformed image)

- `quality` - compression parameter for transformations.

For now list is very short, but it will be extended in future:

### Fit

The image is resized so that it takes up as much space as possible within a bounding box defined by the given width and height parameters.
The original aspect ratio is retained and all of the original image is visible.

### Fill

Fills image to given width & height.

## Running with docker

```bash
docker run kexpress/louis
```


### Command line arguments

```bash
./louis --env=<default: .env | path to file with environment variables> \
        --transforms-path=<default: ensure-transforms.json | path to file containing json description of transforms>
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
| `S3_REGION` | Region where S3 is stored |  | Yes |
| `S3_ACCESS_KEY_ID` | Your S3 access key ID |  | Yes |
| `S3_SECRET_ACCESS_KEY` | Your S3 secret key |  | Yes |
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

### Testing

As you remember louis uses some databases for storing data. And they are needed during tests. You can easily run them from `docker-compose`:

```bash
cd build
docker-compose up -d
```

## Monitoring with Prometheus

Metrics are exposed in port `8001` and route `/metrics`