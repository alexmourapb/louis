package louis

import (
	"bytes"
	"encoding/json"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/KazanExpress/louis/pkg/queue"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/streadway/amqp"
	"io"
	"log"
	"mime/multipart"
	"strings"
	"time"

	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const (
	pathToTestDB   = "../../../test/data/test2.db"
	amqpConnection = "amqp://guest:guest@localhost:5672/"
)

func TestUploadAuthorization(test *testing.T) {

	godotenv.Load("../../../.env")

	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")

	request, err := newFileUploadRequest("http://localhost:8000/upload", nil, "file", path)
	failIfError(test, err, "failed to create file upload request")

	response := httptest.NewRecorder()

	appCtx, err := getAppContext()
	defer appCtx.Close()

	failIfError(test, err, "failed to get app ctx")

	UploadHandler(appCtx)(response, request)
	if response.Code != http.StatusUnauthorized {
		test.Fatalf("Response code was %v; want 401", response.Code)
	}
}

func TestClaimAuthorization(test *testing.T) {
	godotenv.Load("../../../.env")

	request, err := newClaimRequest("http://localhost:8000/claim", nil)
	failIfError(test, err, "failed to create file upload request")

	appCtx, err := getAppContext()
	defer appCtx.Close()

	failIfError(test, err, "failed to get app ctx")

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
	failIfError(test, err, "failed to create file upload request")

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))

	appCtx, err := getAppContext()
	defer appCtx.Close()

	failIfError(test, err, "failed to get app ctx")

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

	appCtx, err := getAppContext()
	defer appCtx.Close()

	failIfError(t, err, "failed to get app ctx")

	// Upload request
	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")
	request, err := newFileUploadRequest("http://localhost:8000/upload", nil, "file", path)

	failIfError(t, err, "failed to create file upload request")

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))

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

	imageURL := payload["url"].(string)
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

	ensureDatabaseStateAfterClaim(t, appCtx, imageKey)

	if appCtx.TransformationsEnabled {
		ch, err := appCtx.RabbitMQConnection.Channel()
		defer ch.Close()
		failIfError(t, err, "failed to create amqp channel")

		q, err := queue.DeclareQueue(TransformationsQueueName, ch)
		failIfError(t, err, "failed to create amqp queue")

		msgs, err := queue.Consume(ch, q.Name)
		failIfError(t, err, "failed to consume messages")

		select {
		case d := <-msgs:
			var data ImageData

			defer d.Ack(false)
			failIfError(t, json.Unmarshal(d.Body, &data), "failed to unmarshall amqp message")
			if data.Key != imageKey {
				t.Fatalf("amqp message is invalid: expected %v but get %v", imageKey, data.Key)
			}

			if data.Url != imageURL {
				t.Fatalf("amqp message is invalid: expected %v but get %v", imageURL, data.Url)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("timeout error: rabbitmq message not recieved in 10 seconds")
		}

	}
	// testing if
	// TODO: add more checks(database, rabbitmq, etc)
}

func ensureDatabaseStateAfterClaim(t *testing.T, appCtx *AppContext, imageKey string) {
	// testing if db is in correct state

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
}

func getAppContext() (*AppContext, error) {
	var err error
	appCtx := &AppContext{TransformationsEnabled: true}

	amqpCon := amqpConnection
	if envCon, envIsSet := os.LookupEnv("RABBITMQ_CONNECTION"); envIsSet {
		amqpCon = envCon
	}
	if appCtx.RabbitMQConnection, err = amqp.Dial(amqpCon); err != nil {
		return nil, err
	}

	if appCtx.DB, err = storage.Open(pathToTestDB); err != nil {
		return nil, err
	}

	if err = appCtx.DB.InitDB(); err != nil {
		return nil, err
	}

	return appCtx, nil
}

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

func failIfError(t *testing.T, err error, msg string) {
	if err != nil {
		log.Fatalf("%s - %v", msg, err)
	}
}
