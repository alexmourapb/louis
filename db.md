# Database structure

We use sqlite for storing internal data.

## Account

| ID | PublicKey | SecretKey |
|:--:|:---------:|:---------:|

## Images

| ID | Key | AccountID | URL | Approved | TransformsUploaded | CreateDate | ApproveDate | TransformsUploadDate |
|:--:|:---:|:---------:|:---:|:--------:|:------------------:|:----------:|:-----------:|:--------------------:|


## Transformations

| ID | Name | Tag | Type | Quality | Width | Height |
|:--:|:----:|:---:|:----:|---------|-------|--------|

## ImagesTags

| ImageID | Tag |
|:-------:|:---:|