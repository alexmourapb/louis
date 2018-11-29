# API docs

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
