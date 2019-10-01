package storage

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/suite"

	"github.com/KazanExpress/louis/internal/pkg/utils"
)

type S3TestSuite struct {
	suite.Suite
	ctx *S3Context
}

func TestS3(t *testing.T) {
	var s3suite = new(S3TestSuite)
	suite.Run(t, s3suite)
}

func (s *S3TestSuite) SetupSuite() {

	var cfg = utils.InitConfigFrom("../../../.env")
	var err error
	s.ctx, err = InitS3Context(cfg)
	if err != nil {
		s.T().Fatalf("failed to init s3 context: %v", err)
	}
}

func (s *S3TestSuite) TestUploadFile() {

	filename := "../../../test/data/picture.jpg"

	f, ferr := os.Open(filename)
	if ferr != nil {
		s.T().Fatalf("failed to open file %q, %v", filename, ferr)
	}
	defer f.Close()

	_, err := s.ctx.UploadFile(f, "test/picture.jpg")

	if err != nil {
		s.T().Fatalf("failed to upload image: %v", err)
	}

	_ = s.ctx.DeleteFolder("test")
}

func (s *S3TestSuite) TestGetObjectNotExist() {
	var _, err = s.ctx.GetObject("kek")
	s.Equal(NoSuchKeyError, err, "there should not be object with such key")
}

func (s *S3TestSuite) TestGetObject() {

	var filename = "../../../test/data/picture.jpg"
	var key = "test/picture.jpg"

	f, ferr := os.Open(filename)
	if ferr != nil {
		s.T().Fatalf("failed to open file %q, %v", filename, ferr)
	}
	defer f.Close()

	_, err := s.ctx.UploadFile(f, key)

	if err != nil {
		s.T().Fatalf("failed to upload image: %v", err)
	}

	body, err := s.ctx.GetObject(key)
	s.NoError(err, "object should exist")

	expectedBody, err := ioutil.ReadFile(filename)
	s.NoError(err)
	s.ElementsMatch(expectedBody, body, "content should be the same content")

	_ = s.ctx.DeleteFolder("test")

}

func (s *S3TestSuite) TestDeleteFolder() {

	file1 := "test/pict1.jpg"
	file2 := "test/pict2.jpg"

	_ = godotenv.Load("../../../.env")

	filename := "../../../test/data/picture.jpg"

	// upload file1
	f, ferr := os.Open(filename)
	s.NoError(ferr, "file should be openable")

	_, err := s.ctx.UploadFile(f, file1)
	s.NoError(err, "file1 failed to upload")

	_, err = s.ctx.UploadFile(f, file2)
	s.NoError(err, "file2 failed to upload")

	err = s.ctx.DeleteFolder("test")
	s.NoError(err, "deletion should be successful")

	objects, err := s.ctx.ListFiles("test")
	s.NoError(err, "listing objects should be successful")

	s.Equal(0, len(objects))

	_ = s.ctx.DeleteFolder("test")

}

func (s *S3TestSuite) TestCopyObject() {

	var filename = "../../../test/data/picture.jpg"
	var fileKey = "test/original.jpg"
	var copyFileKey = "test/copy.jpg"

	_ = godotenv.Load("../../../.env")

	f, ferr := os.Open(filename)
	s.NoError(ferr, "should be able to open file")

	defer s.ctx.DeleteFolder("test")

	_, err := s.ctx.UploadFile(f, fileKey)
	s.NoError(err, "should be uploaded successfully")

	err = s.ctx.CopyObject(fileKey, copyFileKey)
	s.NoError(err, "should be copied successfully")

	_ = s.ctx.DeleteFolder("test")
}
