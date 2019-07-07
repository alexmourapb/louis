package storage

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestUploadFile(t *testing.T) {
	godotenv.Load("../../../.env")

	filename := "../../../test/data/picture.jpg"

	f, ferr := os.Open(filename)
	if ferr != nil {
		t.Fatalf("failed to open file %q, %v", filename, ferr)
	}

	_, err := UploadFile(f, "test/picture.jpg")

	if err != nil {
		t.Fatalf("failed to upload image: %v", err)
	}

	// TODO: add cleanup
}

func TestDeleteFolder(t *testing.T) {
	assert := assert.New(t)

	file1 := "test-dir/pict1.jpg"
	file2 := "test-dir/pict2.jpg"

	godotenv.Load("../../../.env")

	filename := "../../../test/data/picture.jpg"

	// upload file1
	f, ferr := os.Open(filename)
	assert.NoError(ferr)

	_, err := UploadFile(f, file1)
	assert.NoError(err)

	_, err = UploadFile(f, file2)
	assert.NoError(err)

	err = DeleteFolder("test-dir")

	assert.NoError(err)

	ensureFilesDeleted(t, "test-dir")

}

func TestCopyObject(t *testing.T) {
	assert := assert.New(t)

	var filename = "../../../test/data/picture.jpg"
	var fileKey = "copy-test-dir/original.jpg"
	var copyFileKey = "copy-test-dir/copy.jpg"

	godotenv.Load("../../../.env")

	f, ferr := os.Open(filename)
	assert.NoError(ferr, "should be able to open file")

	defer DeleteFolder("copy-test-dir")

	_, err := UploadFile(f, fileKey)
	assert.NoError(err, "should be uploaded successfully")

	err = CopyObject(fileKey, copyFileKey)
	assert.NoError(err, "should be copied successfully")

}

func ensureFilesDeleted(t *testing.T, prefix string) {
	sess := getSession()

	svc := s3.New(sess)
	objects, err := svc.ListObjects(&s3.ListObjectsInput{
		Bucket: getBucket(),
		Prefix: aws.String(prefix),
	})

	assert.NoError(t, err)

	assert.Equal(t, 0, len(objects.Contents))
}
