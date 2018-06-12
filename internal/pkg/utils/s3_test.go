package utils

import (
	"github.com/joho/godotenv"
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
