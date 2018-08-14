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
	"github.com/stretchr/testify/suite"
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

type Suite struct {
	suite.Suite
	appCtx *AppContext
}

func (s *Suite) SetupSuite() {
	log.Printf("Executing setup all suite")
	appCtx := &AppContext{}
	appCtx.Config = config.InitFrom("../../../.env")

	s.appCtx = appCtx
	appCtx.WithWork()
}

func (s *Suite) BeforeTest(tn, sn string) {

	var err error
	if s.appCtx.DB, err = storage.Open(s.appCtx.Config); err != nil {
		s.Fail("failed to connect to db - %v", err)
	}

	log.Printf("Executing setup for test")
	if err := s.appCtx.DB.InitDB(); err != nil {
		defer s.Fail("failed to init db - %v", err)
		s.appCtx.DB.DropDB()
		s.appCtx.DropRedis()
	}

}

func (s *Suite) AfterTest(tn, sn string) {
	log.Printf("Executing tear down test")
	s.appCtx.DB.DropDB()
	s.appCtx.DB.Close()
	s.appCtx.DropRedis()
}

func (s *Suite) TearDownSuite() {
	log.Printf("Executing tear down all")
	s.appCtx.Pool.Drain()
	s.appCtx.Pool.Stop()
}

func TestEndpointSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestClaimAuthorization() {
	appCtx := s.appCtx

	request, err := newClaimRequest("http://localhost:8000/claim", nil)
	s.NoError(err)

	response := httptest.NewRecorder()
	ClaimHandler(appCtx)(response, request)

	s.Equal(http.StatusUnauthorized, response.Code, "should respond with 401")

	// appCtx.DropAll()

}
func (s *Suite) TestUploadAuthorization() {

	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")

	request, err := newFileUploadRequest("http://localhost:8000/upload", nil, "file", path)
	s.NoError(err, "failed to create file upload request")

	response := httptest.NewRecorder()

	UploadHandler(s.appCtx)(response, request)

	s.Equal(http.StatusUnauthorized, response.Code, "should respond with 401")

	// appCtx.DropAll()

}

func (s *Suite) TestUpload() {
	gmega := gomega.NewGomegaWithT(s.T())

	assert := assert.New(s.T())
	appCtx := s.appCtx

	appCtx.Config.CleanUpDelay = 0
	appCtx = appCtx.WithWork()

	path, _ := os.Getwd()
	path = filepath.Join(path, "./../../../test/data/picture.jpg")
	request, err := newFileUploadRequest("http://localhost:8000/upload", nil, "file", path)
	assert.NoError(err, "failed to create file upload request")

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))

	response := httptest.NewRecorder()
	UploadHandler(appCtx)(response, request)

	assert.Equal(http.StatusOK, response.Code, "should respond with 200")

	var resp ResponseTemplate
	assert.NoError(json.Unmarshal(response.Body.Bytes(), &resp), "failed to unmarshall response body")

	assert.Empty(resp.Error, "expected response error to be empty")

	assert.NotNil(resp.Payload)

	var payload = resp.Payload.(map[string]interface{})

	url := payload["originalUrl"].(string)
	imageKey := payload["key"].(string)

	if !strings.HasPrefix(url, "http") || !strings.HasSuffix(url, ".jpg") {
		s.Failf("url should start with http(s?):// and end with .jpg but recieved - %v", url)
	}

	img, err := appCtx.DB.QueryImageByKey(imageKey)
	assert.NoError(err)

	assert.Equal(url, img.URL, "url from response and in database should be the same")

	gmega.Eventually(func() bool {
		img, err := appCtx.DB.QueryImageByKey(imageKey)
		if err != nil {
			log.Printf("TEST ERROR: %v", err)
		}
		return err == nil && img.Deleted

	}, 10, 1).Should(gomega.BeTrue())

	// appCtx.DropAll()
}

func (s *Suite) TestUploadWithName() {
	const mykey = "my_key"

	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")
	request, err := newFileUploadRequest("http://localhost:8000/upload", map[string]string{"tags": " tag1 , tag2 , super-tag", "key": mykey}, "file", path)
	s.NoError(err)

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))
	response := httptest.NewRecorder()
	UploadHandler(s.appCtx)(response, request)

	s.Equal(http.StatusOK, response.Code, "should respond with 200 OK")
	var resp ResponseTemplate

	s.NoError(json.Unmarshal(response.Body.Bytes(), &resp))
	s.Empty(resp.Error)

	s.NotNil(resp.Payload)

	var payload = resp.Payload.(map[string]interface{})
	s.Equal(mykey, payload["key"])
}

func (s *Suite) TestUploadWithNameFail() {
	const mykey = "my_key"

	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")
	request, err := newFileUploadRequest("http://localhost:8000/upload", map[string]string{"tags": " tag1 , tag2 , super-tag", "key": mykey}, "file", path)
	s.NoError(err)

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))
	response := httptest.NewRecorder()
	UploadHandler(s.appCtx)(response, request)

	s.Equal(http.StatusOK, response.Code, "should respond with 200 OK")

	response = httptest.NewRecorder()
	UploadHandler(s.appCtx)(response, request)

	s.Equal(http.StatusBadRequest, response.Code)

}

func (s *Suite) TestUploadWithTags() {

	assert := assert.New(s.T())

	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")
	request, err := newFileUploadRequest("http://localhost:8000/upload", map[string]string{"tags": " tag1 , tag2 , super-tag"}, "file", path)
	failIfError(s.T(), err, "failed to create file upload request")

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))

	response := httptest.NewRecorder()
	UploadHandler(s.appCtx)(response, request)

	assert.Equal(http.StatusOK, response.Code, "should respond with 200")

	var resp ResponseTemplate
	failIfError(s.T(), json.Unmarshal(response.Body.Bytes(), &resp), "failed to unmarshall response body")

	assert.Empty(resp.Error, "expected response error to be empty")

	assert.NotNil(resp.Payload)
	ensureTransformations(s.T(), s.appCtx, resp)

}

func (s *Suite) TestClaim() {

	assert := assert.New(s.T())
	assert.NoError(s.appCtx.DB.EnsureTransformations(tlist))

	// Upload request
	path, _ := os.Getwd()
	path = filepath.Join(path, "../../../test/data/picture.jpg")
	request, err := newFileUploadRequest("http://localhost:8000/upload", map[string]string{"tags": "thubnail_small_low"}, "file", path)

	failIfError(s.T(), err, "failed to create file upload request")

	request.Header.Add("Authorization", os.Getenv("LOUIS_PUBLIC_KEY"))

	response := httptest.NewRecorder()
	UploadHandler(s.appCtx)(response, request)

	assert.Equal(http.StatusOK, response.Code, "should respond with 200")

	var resp ResponseTemplate
	failIfError(s.T(), json.Unmarshal(response.Body.Bytes(), &resp), "failed to unmarshall response body")

	assert.Empty(resp.Error, "expected response error to be empty")

	var payload = resp.Payload.(map[string]interface{})

	// imageURL := payload["url"].(string)
	imageKey := payload["key"].(string)

	// Claim response testing
	response = httptest.NewRecorder()
	request, err = newClaimRequest("http://localhost:8000/claim", map[string]interface{}{"keys": []string{imageKey}})

	failIfError(s.T(), err, "failed to create claim request")
	request.Header.Add("Authorization", os.Getenv("LOUIS_SECRET_KEY"))

	ClaimHandler(s.appCtx)(response, request)

	assert.Equal(http.StatusOK, response.Code, "should respond with 200")

	ensureDatabaseStateAfterClaim(s.T(), s.appCtx, imageKey)

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

func ensureDatabaseStateAfterClaim(t *testing.T, appCtx *AppContext, imageKey string) {
	// testing if db is in correct state

	img, err := appCtx.DB.QueryImageByKey(imageKey)
	assert.NoError(t, err)

	assert.True(t, img.Approved)
}

func getAppContext() (*AppContext, error) {
	var err error
	appCtx := &AppContext{}
	appCtx.Config = config.InitFrom("../../../.env")

	if appCtx.DB, err = storage.Open(appCtx.Config); err != nil {
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
