package louis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/KazanExpress/Louis/internal/pkg/storage"
	"github.com/rs/xid"
	image2 "image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const (
	MaxImageSize       = 5 * 1024 * 1024 // bytes
	CompressionQuality = 20
)

type ImageData struct {
	Key string `json:"key"`
	Url string `json:"url"`
}

type ImageKey struct {
	Key string `json:"key"`
}

type ResponseTemplate struct {
	Error   string      `json:"error"`
	Payload interface{} `json:"payload"`
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

func UploadHandler(db *storage.DB) http.HandlerFunc {
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

		image, _, err := image2.Decode(bytes.NewReader(buffer.Bytes()))
		if failOnError(w, err, "error on creating an Image object from bytes", http.StatusBadRequest) {
			return
		}

		buffer = bytes.Buffer{}
		err = jpeg.Encode(&buffer, image, &jpeg.Options{Quality: CompressionQuality})
		if failOnError(w, err, "error on compressing an image", http.StatusBadRequest) {
			return
		}

		var imageData ImageData
		imageData.Key = xid.New().String()

		tx, err := db.Begin()
		if failOnError(w, err, "error on creating transaction", http.StatusInternalServerError) {
			return
		}

		_, err = tx.CreateImage(imageData.Key, userID)
		if failOnError(w, err, "error on executing transaction", http.StatusInternalServerError) {
			return
		}

		if failOnError(w, tx.Commit(), "error on commiting create image transaction", http.StatusInternalServerError) {
			return
		}

		output, err := storage.UploadFile(bytes.NewReader(buffer.Bytes()), imageData.Key+".jpg")
		if failOnError(w, err, "failed to upload compressed image", http.StatusInternalServerError) {
			return
		}

		tx, err = db.Begin()
		if failOnError(w, err, "error on creating transaction", http.StatusInternalServerError) {
			return
		}

		if failOnError(w, tx.SetImageURL(imageData.Key, userID, output.Location), "error on executing transaction", http.StatusInternalServerError) {
			return
		}

		if failOnError(w, tx.Commit(), "error on commiting set image URL transaction", http.StatusInternalServerError) {
			return
		}

		imageData.Url = output.Location
		respondWithJson(w, "", imageData, 200)
	})
}

func ClaimHandler(db *storage.DB) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err, _ := authorizeBySecretKey(r.Header.Get("Authorization"))
		if err != nil {
			respondWithJson(w, err.Error(), nil, http.StatusUnauthorized)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			respondWithJson(w, err.Error(), nil, http.StatusBadRequest)
			return
		}
		var imageKey ImageKey
		err = json.Unmarshal(body, &imageKey)
		if err != nil {
			respondWithJson(w, err.Error(), nil, http.StatusBadRequest)
		}
		// here we should create transformations for image key

	})
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
