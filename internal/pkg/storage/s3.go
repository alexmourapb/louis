package storage

import (
	"bytes"
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"os"
)

// UploadFile - uploads the file with objectKey key
func UploadFile(file io.Reader, objectKey string) (string, error) {

	sess := getSession()

	manager := s3manager.NewUploader(sess)
	out, err := manager.Upload(&s3manager.UploadInput{
		Bucket: getBucket(),
		Body:   file,
		Key:    aws.String(objectKey),
		ACL:    aws.String("public-read"),
		// We assume that image is already converted to jpg
		ContentType: aws.String("image/jpeg"),
	})

	return out.Location, err
}

// UploadFileWithContext - uploads the file with objectKey key with context
func UploadFileWithContext(ctx context.Context, file io.Reader, objectKey string) (string, error) {

	sess := getSession()

	manager := s3manager.NewUploader(sess)
	out, err := manager.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: getBucket(),
		Body:   file,
		Key:    aws.String(objectKey),
		ACL:    aws.String("public-read"),
		// We assume that image is already converted to jpg
		ContentType: aws.String("image/jpeg"),
	})

	return out.Location, err
}

func getBucket() *string {
	return aws.String(os.Getenv("S3_BUCKET"))
}

func getSession() *session.Session {
	return session.Must(session.NewSession(&aws.Config{
		Endpoint: aws.String(os.Getenv("S3_ENDPOINT")),
		LogLevel: aws.LogLevel(aws.LogDebugWithHTTPBody),
	}))
}

// DeleteFolder - Deletes all files with given prefix
func DeleteFolder(prefix string) error {
	sess := getSession()

	svc := s3.New(sess)

	svc.Handlers.Build.PushBack(func(r *request.Request) {

		if r.Operation.Name == "DeleteObjects" {
			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body)
			updated := bytes.Replace(buf.Bytes(), []byte(` xmlns="http://s3.amazonaws.com/doc/2006-03-01/"`), []byte(""), -1)
			r.SetReaderBody(bytes.NewReader(updated))
		}
	})

	objects, err := svc.ListObjects(&s3.ListObjectsInput{
		Bucket: getBucket(),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		return err
	}
	obIdentifiers := make([]*s3.ObjectIdentifier, len(objects.Contents))
	for i, obj := range objects.Contents {
		obIdentifiers[i] = &s3.ObjectIdentifier{Key: obj.Key}
	}

	_, err = svc.DeleteObjects(&s3.DeleteObjectsInput{
		Bucket: getBucket(),
		Delete: &s3.Delete{
			Objects: obIdentifiers,
		},
	})
	return err
}
