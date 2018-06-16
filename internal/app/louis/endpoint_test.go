package louis

import (
	"bytes"
	"encoding/json"
	"github.com/joho/godotenv"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func newFileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err
}

func newClaimRequest(uri string, payload interface{}) (*http.Request, error) {
	body := &bytes.Buffer{}

	jsonResponse, merror := json.Marshal(payload)
	if merror != nil {
		return nil, merror
	}
	body.Write(jsonResponse)

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Set("Content-Type", "application/json")
	return req, err
}

func TestUploadAuthorization(test *testing.T) {

	godotenv.Load("../../../.env")

	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")

	request, err := newFileUploadRequest("http://localhost:8000/upload", nil, "file", path)
	if err != nil {
		test.Fatalf("Error should be nil but %v", err)
	}
	response := httptest.NewRecorder()
	UploadHandler(nil)(response, request)
	if response.Code != http.StatusUnauthorized {
		test.Fatalf("Response code was %v; want 401", response.Code)
	}
}

func TestClaimAuthorization(test *testing.T) {
	godotenv.Load("../../../.env")

	request, err := newClaimRequest("http://localhost:8000/claim", nil)
	if err != nil {
		test.Fatalf("Failed to create request: %v", err)
	}

	response := httptest.NewRecorder()
	Claim(response, request)
	if response.Code != http.StatusUnauthorized {
		test.Fatalf("Response code was %v; want 401", response.Code)
	}
}

func TestUpload(test *testing.T) {
	godotenv.Load("../../../.env")

	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")
	request, err := newFileUploadRequest("http://localhost:8000/upload", nil, "file", path)
	if err != nil {
		test.Error("Error should not be nil")
		return
	}

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))

	response := httptest.NewRecorder()
	UploadHandler(nil)(response, request)
	if response.Code != http.StatusOK {
		test.Errorf("Response code was %v; want 200", response.Code)
	}
}
