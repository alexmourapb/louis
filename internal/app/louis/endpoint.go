package louis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/KazanExpress/louis/internal/pkg/transformations"
	"github.com/lib/pq"
	"github.com/rs/xid"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

const (
	OriginalTransformName = "original"
)

type ImageData struct {
	Keys []string `json:"keys"`
	URL  string   `json:"url"`
}

type ImageKey struct {
	Key string `json:"key"`
}

type ResponseTemplate struct {
	Error   string      `json:"error"`
	Payload interface{} `json:"payload"`
}

type UploadResponsePayload struct {
	ImageKey        string            `json:"key"`
	OriginalURL     string            `json:"originalUrl"`
	Transformations map[string]string `json:"transformations"`
}

func GetDashboard(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, here is your dashboard")
}

func GetHealth(appCtx *AppContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		health := GetHealthStats()
		body, _ := json.Marshal(health)
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	})
}

func failOnError(w http.ResponseWriter, err error, logMessage string, code int) (failed bool) {
	if err != nil {
		if logMessage != "" {
			log.Printf("ERROR: %s - %v", logMessage, err)
		}
		respondWithJSON(w, err.Error(), nil, code)
		return true
	}
	return false
}

func makeTransformsPayload(imgKey string, transformationsURLs map[string]string) UploadResponsePayload {
	return UploadResponsePayload{
		ImageKey:        imgKey,
		OriginalURL:     transformationsURLs[OriginalTransformName],
		Transformations: transformationsURLs,
	}
}

func makePath(transformName, imageKey string) string {
	return fmt.Sprintf("%s/%s.jpg", imageKey, transformName)
}

func (appCtx *AppContext) uploadPictureAndTransforms(imgID int64, imgKey string, buffer *[]byte) (map[string]string, error) {
	trans, err := appCtx.DB.GetTransformations(imgID)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var ers = make(chan error, len(trans)+1)
	var transformURLs = make(map[string]string)

	ctx, cancelCtx := context.WithCancel(context.Background())
	wg.Add(1 + len(trans))
	go func(ctx context.Context) {
		defer wg.Done()

		transformURLs[OriginalTransformName], err = storage.UploadFileWithContext(ctx, bytes.NewReader(*buffer), makePath(OriginalTransformName, imgKey))
		if err != nil {
			ers <- err
		}
	}(ctx)

	for _, tr := range trans {
		switch tr.Type {
		case "fit":
			go func(ctx context.Context, tran storage.Transformation) {

				defer wg.Done()

				result, err := transformations.Fit(*buffer, tran.Width, tran.Quality)
				if err != nil {
					ers <- err
					return
				}
				transformURLs[tran.Name], err = storage.UploadFileWithContext(ctx, bytes.NewReader(result), makePath(tran.Name, imgKey))
				if err != nil {
					ers <- err
				}
			}(ctx, tr)

			break
		case "fill":
			go func(ctx context.Context, tran storage.Transformation) {
				defer wg.Done()

				result, err := transformations.Fill(*buffer, tran.Width, tran.Height, tran.Quality)
				if err != nil {
					ers <- err
					return
				}
				transformURLs[tran.Name], err = storage.UploadFileWithContext(ctx, bytes.NewReader(result), makePath(tran.Name, imgKey))
				if err != nil {
					ers <- err
				}
			}(ctx, tr)
			break
		default:
			wg.Done()
		}
	}
	select {
	case er := <-ers:
		cancelCtx()
		log.Printf("ERROR: on parallel transnforms - %v", er)
		return nil, er

	case <-func() chan bool {
		wg.Wait()
		ch := make(chan bool, 1)
		ch <- true
		return ch
	}():
		err := appCtx.DB.SetTransformsUploaded(imgID)
		if err != nil {
			log.Printf("ERROR: failed to mark image as transformed - %v", err)
		}
		return transformURLs, err
	}
}

func (appCtx *AppContext) parseAndUpload(w http.ResponseWriter, r *http.Request, userID int32) (returnedError bool, transformsURLs map[string]string, imgKey string) {

	returnedError = true

	r.ParseMultipartForm(appCtx.Config.MaxImageSize)
	file, _, err := r.FormFile("file")
	defer file.Close()

	if failOnError(w, err, "error on reading file from multipart", http.StatusBadRequest) {
		return
	}

	tagsStr := strings.Replace(r.FormValue("tags"), " ", "", -1)
	var tags []string
	if tagsStr != "" {

		tags = strings.Split(tagsStr, ",")
		for _, tag := range tags {
			if len(tag) > storage.TagLength {
				respondWithJSON(w, fmt.Sprintf("tag should not be longer than %v", storage.TagLength), nil, http.StatusBadRequest)
				return
			}
		}
	}

	var buffer bytes.Buffer
	io.Copy(&buffer, file)
	bufferBytes := buffer.Bytes()

	_, _, err = image.Decode(bytes.NewReader(bufferBytes))
	if failOnError(w, err, "error on creating an Image object from bytes", http.StatusBadRequest) {
		return
	}
	imgKey = xid.New().String()

	keyArg := r.FormValue("key")
	if keyArg != "" {
		imgKey = keyArg
	}

	imgID, err := appCtx.DB.AddImage(imgKey, userID, tags...)
	if err != nil {

		if pger, ok := err.(*pq.Error); ok && pger.Constraint == "images_key_key" {
			failOnError(w, err, "image with such key is already exists", http.StatusBadRequest)
		} else {
			failOnError(w, err, "error on creating db record", http.StatusInternalServerError)
		}
		return
	}

	transformsURLs, err = appCtx.uploadPictureAndTransforms(imgID, imgKey, &bufferBytes)

	if failOnError(w, err, "failed to upload transforms", http.StatusInternalServerError) {
		return
	}
	return false, transformsURLs, imgKey
}

func UploadHandler(appCtx *AppContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		userID, err := authorizeByPublicKey(r.Header.Get("Authorization"))
		if err != nil {
			respondWithJSON(w, err.Error(), nil, http.StatusUnauthorized)
			return
		}

		returnedError, transformsURLs, imgKey := appCtx.parseAndUpload(w, r, userID)
		if returnedError {
			return
		}

		_, err = appCtx.Enqueuer.EnqueueUniqueIn(CleanupTask, int64(appCtx.Config.CleanUpDelay*60), map[string]interface{}{"key": imgKey})
		if err != nil {
			log.Printf("ERROR: failed to enqueue clean up task: %v", err)
		}

		if failOnError(w, setImageURL(appCtx, imgKey, transformsURLs[OriginalTransformName], userID), "failed to set image url", http.StatusInternalServerError) {
			return
		}

		log.Printf("INFO: image with key %v and %v transforms uploaded", imgKey, len(transformsURLs))
		respondWithJSON(w, "", makeTransformsPayload(imgKey, transformsURLs), 200)
	})
}

func ClaimHandler(appCtx *AppContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := authorizeBySecretKey(r.Header.Get("Authorization"))
		if err != nil {
			respondWithJSON(w, err.Error(), nil, http.StatusUnauthorized)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("ERROR: error on reading request body - %v", err)
			respondWithJSON(w, err.Error(), nil, http.StatusBadRequest)
			return
		}

		var img ImageData
		err = json.Unmarshal(body, &img)
		if err != nil {
			log.Printf("ERROR: error on object deserialization - %v", err)
			respondWithJSON(w, err.Error(), nil, http.StatusBadRequest)
			return
		}

		if img.Keys == nil || len(img.Keys) == 0 {
			log.Printf("ERROR: keys not provided")
			respondWithJSON(w, "no keys provided", nil, http.StatusBadRequest)
			return
		}

		images, err := appCtx.DB.GetImagesWithKeys(img.Keys)
		if failOnError(w, err, "failed to get images with keys", http.StatusBadRequest) {
			return
		}

		for _, image := range *images {

			if image.Deleted {
				respondWithJSON(w, fmt.Sprintf("image with key = %v is deleted", image.Key), "", http.StatusBadRequest)
				log.Printf("INFO: trying to claim deleted image")

				return
			}
		}

		if failOnError(w, appCtx.DB.SetClaimImages(img.Keys, userID), "failed to claim image", http.StatusInternalServerError) {
			return
		}

		log.Printf("INFO: images with keys [%v] claimed", img.Keys)
		respondWithJSON(w, "", "ok", 200)
	})
}

func UploadWithClaimHandler(appCtx *AppContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		userID, err := authorizeBySecretKey(r.Header.Get("Authorization"))
		if err != nil {
			respondWithJSON(w, err.Error(), nil, http.StatusUnauthorized)
			return
		}

		returnedError, transformsURLs, imgKey := appCtx.parseAndUpload(w, r, userID)
		if returnedError {
			return
		}

		if failOnError(w, setImageURL(appCtx, imgKey, transformsURLs[OriginalTransformName], userID), "failed to set image url", http.StatusInternalServerError) {
			return
		}

		if failOnError(w, appCtx.DB.SetClaimImage(imgKey, userID), "failed to claim image", http.StatusInternalServerError) {
			return
		}

		log.Printf("INFO: image with key %v and %v transforms uploaded and claimed", imgKey, len(transformsURLs))
		respondWithJSON(w, "", makeTransformsPayload(imgKey, transformsURLs), 200)
	})
}

func setImageURL(appCtx *AppContext, imageKey, url string, userID int32) error {
	return appCtx.DB.SetImageURL(imageKey, userID, url)
}

func respondWithJSON(w http.ResponseWriter, err string, payload interface{}, code int) error {
	response := ResponseTemplate{Error: err, Payload: payload}
	jsonResponse, merror := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")

	if merror != nil {
		log.Printf("ERROR: some shit happened on wrapping response to json. payload: %v", payload)
		http.Error(w, "Failed to construct response", http.StatusInternalServerError)
		return merror
	}

	w.WriteHeader(code)
	_, herr := w.Write(jsonResponse)
	return herr
}

func authorizeByPublicKey(publicKey string) (userID int32, err error) {
	if publicKey == os.Getenv("LOUIS_PUBLIC_KEY") {
		return 1, nil
	}
	return -1, fmt.Errorf("account not found")
}

func authorizeBySecretKey(publicKey string) (userID int32, err error) {
	if publicKey == os.Getenv("LOUIS_SECRET_KEY") {
		return 1, nil
	}
	return -1, fmt.Errorf("account not found")
}
