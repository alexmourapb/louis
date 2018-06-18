package louis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/KazanExpress/Louis/internal/pkg/storage"
	"github.com/KazanExpress/Louis/internal/pkg/utils"
	"github.com/rs/xid"
	image2 "image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const MaxImageSize = 5 * 1024 * 1024 // bytes

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

func UploadHandler(db *storage.DB) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		err, userID := authorizeByPublicKey(r.Header.Get("Authorization"))
		if err != nil {
			respondWithJson(w, err.Error(), nil, http.StatusUnauthorized)
			return
		}

		r.ParseMultipartForm(MaxImageSize)
		file, _, err := r.FormFile("file")

		if err != nil {
			log.Printf("ERROR: error on reading file from multipart - %v", err)
			respondWithJson(w, err.Error(), nil, http.StatusBadRequest)
			return
		}

		defer file.Close()
		var buffer bytes.Buffer
		io.Copy(&buffer, file)

		image, _, err := image2.Decode(bytes.NewReader(buffer.Bytes()))
		if err != nil {
			log.Printf("ERROR: error on creating an Image object from bytes - %v", err)
			respondWithJson(w, err.Error(), nil, http.StatusBadRequest)
			return
		}

		buffer = bytes.Buffer{}
		err = jpeg.Encode(&buffer, image, &jpeg.Options{Quality: 20})
		if err != nil {
			log.Printf("ERROR: error on compressing an image - %v", err)
			respondWithJson(w, err.Error(), nil, http.StatusBadRequest)
			return
		}

		var imageData ImageData
		imageData.Key = xid.New().String()

		tx, err := db.Begin()
		if err != nil {
			log.Printf("ERROR: error on creating transaction - %v", err)
			respondWithJson(w, err.Error(), nil, http.StatusInternalServerError)
			return
		}

		tx.CreateImage(imageData.Key, userID)

		output, err := utils.UploadFile(bytes.NewReader(buffer.Bytes()), imageData.Key+".jpg")
		if err != nil {
			respondWithJson(w, err.Error(), nil, http.StatusInternalServerError)
			return
		}
		// tx.SetImageUrl(imageData.Key, output.Location)
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
