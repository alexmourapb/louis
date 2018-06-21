# Louis

Service for uploading images to any S3 compatible storage.

## Running the server

```bash
$ go build ./cmd/louis
$ ./louis
```

## Environment variables

```
S3_BUCKET=<name of S3 bucket>
AWS_REGION=<region where s3 is stored>
S3_ENDPOINT=<url to S3 api server; if not set used AWS endpoint by default>
AWS_ACCESS_KEY_ID=<your access key id>
AWS_SECRET_ACCESS_KEY=<your secret key>
LOUIS_PUBLIC_KEY=<key used for uploading images>
LOUIS_SECRET_KEY=<key used for claiming images>
DATA_SOURCE_NAME=<path to sqlite db>
S3_BUCKET_ENDPOINT=<url to bucket endpoint>
RABBITMQ_CONNECTION=<connection to rabbitmq, used if transformations enabled>
TRANSFORMATIONS_ENABLED=<true/false; if true then information about claimed images will be passed via rabbitmq to be transformed>
```

Example:

```
S3_BUCKET=my-bucket-name
AWS_REGION=ru-msk
S3_ENDPOINT=https://hb.bizmrg.com
AWS_ACCESS_KEY_ID=super-public-key-id
AWS_SECRET_ACCESS_KEY=super-secret-key
LOUIS_PUBLIC_KEY=well-known-public-key
LOUIS_SECRET_KEY=secret-louis-key
DATA_SOURCE_NAME=mysqlite.db
S3_BUCKET_ENDPOINT=https://my-bucket-name.hb.bizmrg.com/
RABBITMQ_CONNECTION=amqp://guest:guest@localhost:5672/
TRANSFORMATIONS_ENABLED=false
```