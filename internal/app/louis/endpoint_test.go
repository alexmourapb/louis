package louis

import (
	"bytes"
	"encoding/json"
	"github.com/KazanExpress/Louis/internal/pkg/storage"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"log"
	"mime/multipart"
	"strings"

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

var pathToTestDB = "../../../test/data/test.db"

func failIfError(t *testing.T, err error, msg string) {
	if err != nil {
		log.Fatalf("%s - %v", msg, err)
	}
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

	appCtx := &AppContext{}
	appCtx.DB, err = storage.Open(pathToTestDB)

	failIfError(test, err, "failed to open db")
	defer os.Remove(pathToTestDB)
	defer os.Remove(pathToTestDB + "-journal")
	defer appCtx.DB.Close()

	failIfError(test, appCtx.DB.InitDB(), "failed to create initial tables")

	UploadHandler(appCtx)(response, request)
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

	appCtx := &AppContext{}
	appCtx.DB, err = storage.Open(pathToTestDB)

	failIfError(test, err, "failed to open db")
	defer os.Remove(pathToTestDB)
	defer os.Remove(pathToTestDB + "-journal")

	defer appCtx.DB.Close()

	failIfError(test, appCtx.DB.InitDB(), "failed to create initial tables")

	response := httptest.NewRecorder()
	ClaimHandler(appCtx)(response, request)
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

	appCtx := &AppContext{}
	appCtx.DB, err = storage.Open(pathToTestDB)

	failIfError(test, err, "failed to open db")
	defer os.Remove(pathToTestDB)
	defer os.Remove(pathToTestDB + "-journal")

	defer appCtx.DB.Close()

	failIfError(test, appCtx.DB.InitDB(), "failed to create initial tables")

	response := httptest.NewRecorder()
	UploadHandler(appCtx)(response, request)
	if response.Code != http.StatusOK {
		test.Fatalf("Response code was %v; want 200", response.Code)
	}

	var resp ResponseTemplate
	failIfError(test, json.Unmarshal(response.Body.Bytes(), &resp), "failed to unmarshall response body")

	if resp.Error != "" {
		test.Fatalf("expected response error to be empty but get - %s", resp.Error)
	}

	var payload = resp.Payload.(map[string]interface{})

	url := payload["url"].(string)
	imageKey := payload["key"].(string)

	if !strings.HasPrefix(url, "http") || !strings.HasSuffix(url, ".jpg") {
		test.Fatalf("url should start with http(s?):// and end with .jpg but recieved - %v", url)
	}

	rows, err := appCtx.DB.Query("SELECT URL FROM Images WHERE key=?", imageKey)
	defer rows.Close()

	var URL string

	if rows.Next() {
		failIfError(test, rows.Scan(&URL), "failed to scan URL column")
	} else {
		test.Fatalf("image with key %s not found", imageKey)
	}

	if URL != url {
		test.Fatalf("expected URL = true but get %v", URL)
	}
}

func TestClaim(t *testing.T) {
	godotenv.Load("../../../.env")

	// Upload request
	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")
	request, err := newFileUploadRequest("http://localhost:8000/upload", nil, "file", path)
	if err != nil {
		t.Error("Error should not be nil")
		return
	}

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))

	appCtx := &AppContext{}
	appCtx.DB, err = storage.Open(pathToTestDB)

	failIfError(t, err, "failed to open db")
	defer os.Remove(pathToTestDB)
	defer os.Remove(pathToTestDB + "-journal")

	defer appCtx.DB.Close()

	failIfError(t, appCtx.DB.InitDB(), "failed to create initial tables")

	response := httptest.NewRecorder()
	UploadHandler(appCtx)(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("Response code was %v; want 200", response.Code)
	}

	var resp ResponseTemplate
	failIfError(t, json.Unmarshal(response.Body.Bytes(), &resp), "failed to unmarshall response body")

	if resp.Error != "" {
		t.Fatalf("expected response error to be empty but get - %s", resp.Error)
	}

	var payload = resp.Payload.(map[string]interface{})

	// url := payload["url"].(string)
	imageKey := payload["key"].(string)

	// Claim response testing
	response = httptest.NewRecorder()
	request, err = newClaimRequest("http://localhost:8000/claim", resp.Payload)

	failIfError(t, err, "failed to create claim request")
	request.Header.Add("Authorization", os.Getenv("LOUIS_SECRET_KEY"))
	ClaimHandler(appCtx)(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected claim response code 200 bug get %v", response.Code)
	}

	rows, err := appCtx.DB.Query("SELECT Approved FROM Images WHERE key=?", imageKey)
	failIfError(t, err, "failed to execute sql query")

	var approved bool
	if rows.Next() {
		failIfError(t, rows.Scan(&approved), "failed to scan 'approved' value")
	} else {
		t.Fatalf("image with key=%s not found in db", imageKey)
	}

	if !approved {
		t.Fatalf("expected approved to be true but recieved false")
	}
	// TODO: add more checks(database, rabbitmq, etc)
}
