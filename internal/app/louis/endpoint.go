package louis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/KazanExpress/louis/internal/pkg/queue"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/go-redis/redis"
	"github.com/rs/xid"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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

func failOnError(w http.ResponseWriter, err error, logMessage string, code int) (failed bool) {
	if err != nil {
		if logMessage != "" {
			log.Printf("ERROR: %s - %v", logMessage, err)
		}
		respondWithJson(w, err.Error(), nil, code)
		return true
	}
	return false
}

func UploadHandler(appCtx *AppContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		err, userID := authorizeByPublicKey(r.Header.Get("Authorization"))
		if err != nil {
			respondWithJson(w, err.Error(), nil, http.StatusUnauthorized)
			return
		}

		r.ParseMultipartForm(MaxImageSize)
		file, _, err := r.FormFile("file")

		if failOnError(w, err, "error on reading file from multipart", http.StatusBadRequest) {
			return
		}

		defer file.Close()
		var buffer bytes.Buffer
		io.Copy(&buffer, file)

		img, _, err := image.Decode(bytes.NewReader(buffer.Bytes()))
		if failOnError(w, err, "error on creating an Image object from bytes", http.StatusBadRequest) {
			return
		}

		buffer = bytes.Buffer{}
		err = jpeg.Encode(&buffer, img, &jpeg.Options{Quality: HighCompressionQuality})
		if failOnError(w, err, "error on compressing an img", http.StatusBadRequest) {
			return
		}

		var imageData ImageData
		imageData.Key = xid.New().String()

		tx, err := appCtx.DB.Begin()
		if failOnError(w, err, "error on creating transaction", http.StatusInternalServerError) {
			return
		}

		_, err = tx.CreateImage(imageData.Key, userID)
		if failOnError(w, err, "error on executing transaction", http.StatusInternalServerError) {
			return
		}

		if failOnError(w, tx.Commit(), "error on committing create img transaction", http.StatusInternalServerError) {
			return
		}

		output, err := storage.UploadFile(bytes.NewReader(buffer.Bytes()), imageData.Key+".jpg")
		if failOnError(w, err, "failed to upload compressed img", http.StatusInternalServerError) {
			return
		}

		tx, err = appCtx.DB.Begin()
		if failOnError(w, err, "error on creating transaction", http.StatusInternalServerError) {
			return
		}

		if failOnError(w, tx.SetImageURL(imageData.Key, userID, output.Location), "error on executing transaction", http.StatusInternalServerError) {
			return
		}

		if failOnError(w, tx.Commit(), "error on committing set img URL transaction", http.StatusInternalServerError) {
			return
		}

		imageData.URL = output.Location
		respondWithJson(w, "", imageData, 200)
	})
}

func ClaimHandler(appCtx *AppContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err, userID := authorizeBySecretKey(r.Header.Get("Authorization"))
		if err != nil {
			respondWithJson(w, err.Error(), nil, http.StatusUnauthorized)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("ERROR: error on reading request body - %v", err)
			respondWithJson(w, err.Error(), nil, http.StatusBadRequest)
			return
		}

		var img ImageData
		err = json.Unmarshal(body, &img)
		if err != nil {
			log.Printf("ERROR: error on object deserialization - %v", err)
			respondWithJson(w, err.Error(), nil, http.StatusBadRequest)
			return
		}

		img.URL = getURLByImageKey(img.Key)

		if appCtx.TransformationsEnabled {
			if failOnError(w, passImageToAMQP(appCtx, &img), "failed to pass msg to rabbitmq", http.StatusInternalServerError) {
				return
			}
		}
		var buffer = bytes.Buffer{}
		err = downloadFile(img.URL, &buffer)
		if err != nil {
			log.Printf("ERROR: error on downloading image with key '"+img.Key+"' - %v", err)
			respondWithJson(w, err.Error(), nil, http.StatusBadRequest)
			return
		}

		imageHigh, err := jpeg.Decode(bytes.NewReader(buffer.Bytes()))
		if err != nil {
			log.Printf("ERROR: error on decoding an image with original resolution in claim method - %v", err)
			respondWithJson(w, err.Error(), nil, http.StatusInternalServerError)
			return
		}

		var lowBuffer bytes.Buffer
		err = jpeg.Encode(&lowBuffer, imageHigh, &jpeg.Options{Quality: LowCompressionQuality})
		if err != nil {
			log.Printf("ERROR: error on compressing an image in claim method - %v", err)
			respondWithJson(w, err.Error(), nil, http.StatusBadRequest)
			return
		}

		lowImageKey := img.Key + "_low"

		// TODO: Remove the following testing of how low image is saved to S3
		output, err := storage.UploadFile(bytes.NewReader(lowBuffer.Bytes()), lowImageKey+".jpg")
		if failOnError(w, err, "failed to upload compressed image with low quality", http.StatusInternalServerError) {
			return
		}

		tx, err := appCtx.DB.Begin()
		if failOnError(w, err, "failed to create transaction for claiming image", http.StatusInternalServerError) {
			return
		}

		if failOnError(w, tx.ClaimImage(img.Key, userID), "failed to claim image", http.StatusInternalServerError) {
			return
		}

		if failOnError(w, tx.Commit(), "failed to commit claiming image", http.StatusInternalServerError) {
			return
		}

		var imageData ImageData
		imageData.Key = lowImageKey
		imageData.URL = output.Location
		respondWithJson(w, "", imageData, 200)
	})
}

func passImageToAMQP(appCtx *AppContext, image *ImageData) error {

	body, err := json.Marshal(*image)
	if err != nil {
		return err
	}

	return appCtx.Queue.Publish(body)
}

func getURLByImageKey(key string) string {
	// TODO: maybe it's better to get URL from database?
	// at least, it will be useful when we will have
	// separate buckets for each user
	return os.Getenv("S3_BUCKET_ENDPOINT") + key + ".jpg"
}

func downloadFile(url string, w io.Writer) error {

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func respondWithJson(w http.ResponseWriter, err string, payload interface{}, code int) error {
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

func authorizeByPublicKey(publicKey string) (error, int32) {
	if publicKey == os.Getenv("LOUIS_PUBLIC_KEY") {
		return nil, 1
	}
	return fmt.Errorf("account not found"), -1
}

func authorizeBySecretKey(publicKey string) (error, int32) {
	if publicKey == os.Getenv("LOUIS_SECRET_KEY") {
		return nil, 1
	}
	return fmt.Errorf("account not found"), -1
}
