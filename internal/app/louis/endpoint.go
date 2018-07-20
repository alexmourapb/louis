package louis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/KazanExpress/louis/internal/pkg/queue"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/RichardKnop/machinery/v1/backends/result"
	"github.com/go-redis/redis"
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
	"time"
)

const (
	MaxImageSize           = 5 * 1024 * 1024 // bytes
	HighCompressionQuality = 30
	LowCompressionQuality  = 15
)

type AppContext struct {
	DB                     *storage.DB
	Queue                  queue.JobQueue
	TransformationsEnabled bool
	RedisConnection        string
}

type ImageData struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

type ImageKey struct {
	Key string `json:"key"`
}

type ResponseTemplate struct {
	Error   string      `json:"error"`
	Payload interface{} `json:"payload"`
}

func (appCtx *AppContext) DropAll() error {

	if appCtx.TransformationsEnabled {

		client := redis.NewClient(&redis.Options{
			Addr:     appCtx.RedisConnection[8:],
			Password: "", // no password set
			DB:       0,  // use default DB
		})
		err := client.FlushAll().Err()
		if err != nil {
			log.Printf("WARN: failed to drop redis - %v", err)
		}
	}
	return appCtx.DB.DropDB()
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

func UploadHandler(appCtx *AppContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		userID, err := authorizeByPublicKey(r.Header.Get("Authorization"))
		if err != nil {
			respondWithJSON(w, err.Error(), nil, http.StatusUnauthorized)
			return
		}

		r.ParseMultipartForm(MaxImageSize)
		file, _, err := r.FormFile("file")

		if failOnError(w, err, "error on reading file from multipart", http.StatusBadRequest) {
			return
		}

		tagsStr := strings.Replace(r.FormValue("tags"), " ", "", -1)
		var tags []string
		if tagsStr != "" {
			if !appCtx.TransformationsEnabled {
				log.Printf("WARN: transformations disabled. ignoring recived tags")
			} else {

				tags = strings.Split(tagsStr, ",")
				for _, tag := range tags {
					if len(tag) > storage.TagLength {
						respondWithJSON(w, fmt.Sprintf("tag should not be longer than %v", storage.TagLength), nil, http.StatusBadRequest)
						return
					}
				}
			}
		}

		defer file.Close()
		var buffer bytes.Buffer
		io.Copy(&buffer, file)

		_, _, err = image.Decode(bytes.NewReader(buffer.Bytes()))
		if failOnError(w, err, "error on creating an Image object from bytes", http.StatusBadRequest) {
			return
		}

		var imageData ImageData
		imageData.Key = xid.New().String()

		_, err = appCtx.DB.AddImage(imageData.Key, userID, tags...)
		failOnError(w, err, "error on creating db record", http.StatusInternalServerError)

		imageURL, err := storage.UploadFile(bytes.NewReader(buffer.Bytes()), "originals/"+imageData.Key+".jpg")
		if failOnError(w, err, "failed to upload compressed img", http.StatusInternalServerError) {
			return
		}

		failOnError(w, setImageURL(appCtx, imageData.Key, imageURL, userID), "failed to set image url", http.StatusInternalServerError)

		imageData.URL = imageURL
		respondWithJSON(w, "", imageData, 200)
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

		image, err := appCtx.DB.QueryImageByKey(img.Key)
		if failOnError(w, err, "failed to get image by key", http.StatusInternalServerError) {
			return
		}

		if appCtx.TransformationsEnabled {
			if failOnError(w, addImageTransformsTasksToQueue(appCtx, image), "failed to pass msg to rabbitmq", http.StatusInternalServerError) {
				return
			}
		}

		if failOnError(w, appCtx.DB.SetClaimImage(image.Key, userID), "failed to claim image", http.StatusInternalServerError) {
			return
		}

		respondWithJSON(w, "", "ok", 200)
	})
}

func setImageURL(appCtx *AppContext, imageKey, url string, userID int32) error {
	tx, err := appCtx.DB.Begin()
	if err != nil {
		return fmt.Errorf("error on creating transaction - %v", err)
	}

	err = tx.SetImageURL(imageKey, userID, url)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func addImageTransformsTasksToQueue(appCtx *AppContext, image *storage.Image) error {

	// select transformations by image
	trans, err := appCtx.DB.GetTransformations(image.ID)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var aresults []*result.AsyncResult

	for _, tran := range trans {
		var aresult *result.AsyncResult
		switch tran.Type {
		case "fit":
			aresult, err = appCtx.Queue.PublishFitTransform(queue.NewTransformData(image, &tran))

			break
		case "fill":
			aresult, err = appCtx.Queue.PublishFillTransform(queue.NewTransformData(image, &tran))
		}

		if err != nil {
			log.Printf("ERROR: failed to enqueue transform fit task: %v", err)
			return err
		}
		wg.Add(1)
		aresults = append(aresults, aresult)
	}
	var ers = make(chan error, len(trans)+1)
	go func() {
		for _, ares := range aresults {
			go func(ar *result.AsyncResult) {
				_, err = ar.Get(time.Duration(time.Millisecond * 200))
				if err != nil {
					ers <- err
				}
				wg.Done()
			}(ares)
		}
		wg.Wait()
		noErr := fmt.Errorf("no error")
		ers <- noErr
		close(ers)

		hasErrors := false
		for err := range ers {

			if err != noErr {
				log.Printf("WARN: failed to perform transform task - %v", err)
				hasErrors = true
			}
		}
		if hasErrors {
			return
		}

		err := appCtx.DB.SetTransformsUploaded(image.ID)
		if err != nil {
			// it may fail because connection to db may be closed
			log.Printf("WARN: failed to set transforms uploaded for image %v : %v", image.ID, err)
		}
	}()

	return nil
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
