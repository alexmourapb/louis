package louis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/KazanExpress/Louis/internal/pkg/storage"
	"github.com/rs/xid"
	"image/jpeg"
	_ "image/png"
	"image"
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

		tx, err := db.Begin()
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

		tx, err = db.Begin()
		if failOnError(w, err, "error on creating transaction", http.StatusInternalServerError) {
			return
		}

		if failOnError(w, tx.SetImageURL(imageData.Key, userID, output.Location), "error on executing transaction", http.StatusInternalServerError) {
			return
		}

		if failOnError(w, tx.Commit(), "error on committing set img URL transaction", http.StatusInternalServerError) {
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
			log.Printf("ERROR: error on reading request body - %v", err)
			respondWithJson(w, err.Error(), nil, http.StatusBadRequest)
			return
		}

		var imageKey ImageKey
		err = json.Unmarshal(body, &imageKey)
		if err != nil {
			log.Printf("ERROR: error on object deserialization - %v", err)
			respondWithJson(w, err.Error(), nil, http.StatusBadRequest)
			return
		}

		var buffer = bytes.Buffer{}
		err = downloadFile(os.Getenv("S3_BUCKET_ENDPOINT") + imageKey.Key + ".jpg", &buffer)
		if err != nil {
			log.Printf("ERROR: error on downloading image with key '" + imageKey.Key + "' - %v", err)
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

		lowImageKey := imageKey.Key + "_low"

		// TODO: Remove the following testing of how low image is saved to S3
		output, err := storage.UploadFile(bytes.NewReader(lowBuffer.Bytes()), lowImageKey + ".jpg")
		if failOnError(w, err, "failed to upload compressed image with low quality", http.StatusInternalServerError) {
			return
		}

		var imageData ImageData
		imageData.Key = lowImageKey
		imageData.Url = output.Location
		respondWithJson(w, "", imageData, 200)
	})
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
