package louis

import (
	"bytes"
	"encoding/json"
	"github.com/KazanExpress/louis/internal/pkg/queue"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
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

const (
	pathToTestDB    = "../../../test/data/test2.db"
	redisConnection = "redis://127.0.0.1:6379"
)

func TestUploadAuthorization(test *testing.T) {

	godotenv.Load("../../../.env")

	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")

	request, err := newFileUploadRequest("http://localhost:8000/upload", nil, "file", path)
	failIfError(test, err, "failed to create file upload request")

	response := httptest.NewRecorder()

	appCtx, err := getAppContext()
	defer appCtx.DropAll()

	failIfError(test, err, "failed to get app ctx")

	UploadHandler(appCtx)(response, request)

	assert.Equal(test, http.StatusUnauthorized, response.Code, "should respond with 401")
}

func TestClaimAuthorization(test *testing.T) {
	godotenv.Load("../../../.env")

	request, err := newClaimRequest("http://localhost:8000/claim", nil)
	failIfError(test, err, "failed to create file upload request")

	appCtx, err := getAppContext()
	defer appCtx.DropAll()

	failIfError(test, err, "failed to get app ctx")

	response := httptest.NewRecorder()
	ClaimHandler(appCtx)(response, request)

	assert.Equal(test, http.StatusUnauthorized, response.Code, "should respond with 401")
}

func TestUpload(test *testing.T) {

	assert := assert.New(test)
	godotenv.Load("../../../.env")

	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")
	request, err := newFileUploadRequest("http://localhost:8000/upload", nil, "file", path)
	failIfError(test, err, "failed to create file upload request")

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))

	appCtx, err := getAppContext()
	defer appCtx.DropAll()

	failIfError(test, err, "failed to get app ctx")

	response := httptest.NewRecorder()
	UploadHandler(appCtx)(response, request)

	assert.Equal(http.StatusOK, response.Code, "should respond with 200")

	var resp ResponseTemplate
	failIfError(test, json.Unmarshal(response.Body.Bytes(), &resp), "failed to unmarshall response body")

	assert.Empty(resp.Error, "expected response error to be empty")

	var payload = resp.Payload.(map[string]interface{})

	url := payload["url"].(string)
	imageKey := payload["key"].(string)

	if !strings.HasPrefix(url, "http") || !strings.HasSuffix(url, ".jpg") {
		test.Fatalf("url should start with http(s?):// and end with .jpg but recieved - %v", url)
	}

	rows, err := appCtx.DB.Query("SELECT URL FROM Images WHERE key=?", imageKey)
	defer rows.Close()

	var urlFromDB string

	if rows.Next() {
		failIfError(test, rows.Scan(&urlFromDB), "failed to scan URL column")
	} else {
		test.Fatalf("image with key %s not found", imageKey)
	}

	assert.Equal(url, urlFromDB, "url from response and in database should be the same")

}

func TestUploadWithTags(t *testing.T) {
	assert := assert.New(t)

	godotenv.Load("../../../.env")

	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")
	request, err := newFileUploadRequest("http://localhost:8000/upload", map[string]string{"tags": "tag1,tag2,super-tag"}, "file", path)
	failIfError(t, err, "failed to create file upload request")

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))

	appCtx, err := getAppContext()
	defer appCtx.DropAll()

	failIfError(t, err, "failed to get app ctx")

	response := httptest.NewRecorder()
	UploadHandler(appCtx)(response, request)

	assert.Equal(http.StatusOK, response.Code, "should respond with 200")

	var resp ResponseTemplate
	failIfError(t, json.Unmarshal(response.Body.Bytes(), &resp), "failed to unmarshall response body")

	assert.Empty(resp.Error, "expected response error to be empty")

	rows, err := appCtx.DB.Query("SELECT COUNT(*) FROM ImageTags")
	failIfError(t, err, "failed to execute query")

	var cnt int
	if rows.Next() {
		failIfError(t, rows.Scan(&cnt), "failed to scan")
		assert.Equal(3, cnt)
	} else {
		t.Fatal("query returning nothing")
	}

}

func TestClaim(t *testing.T) {
	assert := assert.New(t)
	godotenv.Load("../../../.env")

	appCtx, err := getAppContext()
	defer appCtx.DropAll()

	failIfError(t, err, "failed to get app ctx")

	// Upload request
	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")
	request, err := newFileUploadRequest("http://localhost:8000/upload", nil, "file", path)

	failIfError(t, err, "failed to create file upload request")

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))

	response := httptest.NewRecorder()
	UploadHandler(appCtx)(response, request)

	assert.Equal(http.StatusOK, response.Code, "should respond with 200")

	var resp ResponseTemplate
	failIfError(t, json.Unmarshal(response.Body.Bytes(), &resp), "failed to unmarshall response body")

	assert.Empty(resp.Error, "expected response error to be empty")

	var payload = resp.Payload.(map[string]interface{})

	imageURL := payload["url"].(string)
	imageKey := payload["key"].(string)

	// Claim response testing
	response = httptest.NewRecorder()
	request, err = newClaimRequest("http://localhost:8000/claim", resp.Payload)

	failIfError(t, err, "failed to create claim request")
	request.Header.Add("Authorization", os.Getenv("LOUIS_SECRET_KEY"))
	ClaimHandler(appCtx)(response, request)

	assert.Equal(http.StatusOK, response.Code, "should respond with 200")

	ensureDatabaseStateAfterClaim(t, appCtx, imageKey)

	if appCtx.TransformationsEnabled {

		server := appCtx.Queue.(*queue.MachineryQueue).MachineryServer
		tasks, err := server.GetBroker().GetPendingTasks(queue.QueueName)

		failIfError(t, err, "failed to get pending tasks")

		assert.Equal(1, len(tasks), "there should be 1 task")

		taskArg := tasks[0].Args[0]

		data := []byte(taskArg.Value.(string))
		var img ImageData
		failIfError(t, json.Unmarshal(data, &img), "failed to unmarshal recieved bytes")

		assert.Equal(imageKey, img.Key)
		assert.Equal(imageURL, img.URL)

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
	appCtx.RedisConnection = redisConnection

	if appCtx.Queue, err = queue.NewMachineryQueue(redisConnection); err != nil {
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
