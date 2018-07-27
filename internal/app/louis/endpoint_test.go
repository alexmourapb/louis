package louis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	// "github.com/KazanExpress/louis/internal/pkg/queue"
	"github.com/KazanExpress/louis/internal/pkg/config"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	_ "github.com/mattn/go-sqlite3"
	"github.com/onsi/gomega"
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

var tlist = []storage.Transformation{
	{
		Name:    "super_transform",
		Tag:     "thubnail_small_low",
		Width:   100,
		Height:  100,
		Quality: 40,
		Type:    "fit",
	},
	{
		Name:    "cover",
		Tag:     "cover_wide",
		Type:    "fill",
		Width:   1200,
		Height:  200,
		Quality: 70,
	},
}

func TestUploadAuthorization(test *testing.T) {
	appCtx, err := getAppContext()
	defer appCtx.DropAll()
	assert.NoError(test, err)
	appCtx.Config = config.InitFrom("../../../.env")

	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")

	request, err := newFileUploadRequest("http://localhost:8000/upload", nil, "file", path)
	failIfError(test, err, "failed to create file upload request")

	response := httptest.NewRecorder()

	UploadHandler(appCtx)(response, request)

	assert.Equal(test, http.StatusUnauthorized, response.Code, "should respond with 401")
}

func TestClaimAuthorization(test *testing.T) {
	appCtx, err := getAppContext()
	defer appCtx.DropAll()

	assert.NoError(test, err)
	appCtx.Config = config.InitFrom("../../../.env")

	request, err := newClaimRequest("http://localhost:8000/claim", nil)

	response := httptest.NewRecorder()
	ClaimHandler(appCtx)(response, request)

	assert.Equal(test, http.StatusUnauthorized, response.Code, "should respond with 401")
}

func TestUpload(test *testing.T) {
	gomega.RegisterTestingT(test)

	assert := assert.New(test)
	appCtx, err := getAppContext()

	assert.NoError(err)
	appCtx.Config = config.InitFrom("../../../.env")
	appCtx.Config.CleanUpDelay = 0
	appCtx = appCtx.WithWork()

	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")
	request, err := newFileUploadRequest("http://localhost:8000/upload", nil, "file", path)
	failIfError(test, err, "failed to create file upload request")

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))

	defer appCtx.DropAll()

	response := httptest.NewRecorder()
	UploadHandler(appCtx)(response, request)

	assert.Equal(http.StatusOK, response.Code, "should respond with 200")

	var resp ResponseTemplate
	failIfError(test, json.Unmarshal(response.Body.Bytes(), &resp), "failed to unmarshall response body")

	assert.Empty(resp.Error, "expected response error to be empty")

	var payload = resp.Payload.(map[string]interface{})

	url := payload["originalUrl"].(string)
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
	rows.Close()

	gomega.Eventually(func() bool {
		img, err := appCtx.DB.QueryImageByKey(imageKey)
		return err == nil && img.Deleted

	}, 10, 1).Should(gomega.BeTrue())
}

func TestUploadWithTags(t *testing.T) {
	assert := assert.New(t)

	appCtx, err := getAppContext()
	assert.NoError(err)
	appCtx.Config = config.InitFrom("../../../.env")
	appCtx.WithWork()
	defer appCtx.DropAll()

	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")
	request, err := newFileUploadRequest("http://localhost:8000/upload", map[string]string{"tags": " tag1 , tag2 , super-tag"}, "file", path)
	failIfError(t, err, "failed to create file upload request")

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))

	response := httptest.NewRecorder()
	UploadHandler(appCtx)(response, request)

	assert.Equal(http.StatusOK, response.Code, "should respond with 200")

	var resp ResponseTemplate
	failIfError(t, json.Unmarshal(response.Body.Bytes(), &resp), "failed to unmarshall response body")

	assert.Empty(resp.Error, "expected response error to be empty")

	ensureTags(t, appCtx)

	ensureTransformations(t, appCtx, resp)

}

func TestClaim(t *testing.T) {
	assert := assert.New(t)
	appCtx, err := getAppContext()
	assert.NoError(err)
	appCtx.Config = config.InitFrom("../../../.env")
	appCtx.WithWork()

	defer appCtx.DropAll()
	assert.NoError(appCtx.DB.EnsureTransformations(tlist))

	// Upload request
	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")
	request, err := newFileUploadRequest("http://localhost:8000/upload", map[string]string{"tags": "thubnail_small_low"}, "file", path)

	failIfError(t, err, "failed to create file upload request")

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))

	response := httptest.NewRecorder()
	UploadHandler(appCtx)(response, request)

	assert.Equal(http.StatusOK, response.Code, "should respond with 200")

	var resp ResponseTemplate
	failIfError(t, json.Unmarshal(response.Body.Bytes(), &resp), "failed to unmarshall response body")

	assert.Empty(resp.Error, "expected response error to be empty")

	var payload = resp.Payload.(map[string]interface{})

	// imageURL := payload["url"].(string)
	imageKey := payload["key"].(string)

	// Claim response testing
	response = httptest.NewRecorder()
	request, err = newClaimRequest("http://localhost:8000/claim", resp.Payload)

	failIfError(t, err, "failed to create claim request")
	request.Header.Add("Authorization", os.Getenv("LOUIS_SECRET_KEY"))

	ClaimHandler(appCtx)(response, request)

	assert.Equal(http.StatusOK, response.Code, "should respond with 200")

	ensureDatabaseStateAfterClaim(t, appCtx, imageKey)

}

func ensureTransformations(t *testing.T, appCtx *AppContext, resp ResponseTemplate) {
	var payload = resp.Payload.(map[string]interface{})

	imageKey := payload["key"].(string)
	originalURL := payload["originalUrl"].(string)
	transformations := payload["transformations"].(map[string]interface{})

	img, err := appCtx.DB.QueryImageByKey(imageKey)
	assert.NoError(t, err)

	assert.Equal(t, img.URL, originalURL)

	assert.True(t, img.TransformsUploaded)

	trans, err := appCtx.DB.GetTransformations(img.ID)
	assert.NoError(t, err)

	for _, tran := range trans {
		// transformations.
		tran.Name = ""
		matched, err := regexp.Match(fmt.Sprintf("^http(s?)\\:\\/\\/.*%s.*%s\\.jpg", imageKey, tran.Name), []byte(transformations[tran.Name].(string)))

		assert.NoError(t, err)
		assert.True(t, matched)
	}

}

func ensureTags(t *testing.T, appCtx *AppContext) {
	rows, err := appCtx.DB.Query("SELECT COUNT(*) FROM ImageTags")
	defer rows.Close()
	failIfError(t, err, "failed to execute query")

	var cnt int
	if rows.Next() {
		failIfError(t, rows.Scan(&cnt), "failed to scan")
		assert.Equal(t, 3, cnt)
	} else {
		t.Fatal("query returning nothing")
	}
}

func ensureDatabaseStateAfterClaim(t *testing.T, appCtx *AppContext, imageKey string) {
	// testing if db is in correct state

	rows, err := appCtx.DB.Query("SELECT Approved FROM Images WHERE key=?", imageKey)
	failIfError(t, err, "failed to execute sql query")

	defer rows.Close()
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
	appCtx := &AppContext{}

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
