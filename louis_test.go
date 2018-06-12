package main

import (
	"testing"
	"os"
	"bytes"
	"mime/multipart"
	"path/filepath"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestUpload(test *testing.T) {
	path, _ := os.Getwd()
	path += "/test/data/picture.jpg"
	request, err := newFileUploadRequest("http://localhost:8000/upload", nil, "file", path)
	if err != nil {
		test.Error("Error should not be nil")
		return
	}
	response := httptest.NewRecorder();
	Upload(response, request)
	if response.Code != http.StatusOK {
		test.Errorf("Response code was %v; want 200", response.Code)
	}
}
