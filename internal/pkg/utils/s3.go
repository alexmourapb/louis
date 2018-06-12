package utils

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"log"
	"os"
)

// UploadFile - uploads the file with objectKey key
func UploadFile(file *os.File, objectKey string) (*s3manager.UploadOutput, error) {

	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint: aws.String(os.Getenv("S3_ENDPOINT")),
	}))

	filename := "../../../test/data/picture.jpg"

	f, ferr := os.Open(filename)
	if ferr != nil {
		log.Printf("ERROR: s3 failed to open file %q, %v", filename, ferr)
		return nil, ferr
	}

	manager := s3manager.NewUploader(sess)
	out, err := manager.Upload(&s3manager.UploadInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Body:   f,
		Key:    aws.String(objectKey),
		ACL:    aws.String("public-read"),
		// We assume that image is already converted to jpg
		ContentType: aws.String("image/jpeg"),
	})

	return out, err
}
