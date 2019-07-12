package louis

import (
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
)

const (
	// "original" transform is not really original image,
	// instead, it's transform which does not change dimension size of image
	// and compresses it a bit
	OriginalTransformName    = "original"
	OriginalTransformQuality = 70
	// At some moment, we came up with a need to make new transforms on old images
	// But there is no real copy of uploaded image, only "original" image which lost
	// some quality. So, if we will apply new transforms on "original" transform
	// quality of resulting images will differ from quality of images resulted from
	// applying new transforms for newly uploaded images. By this, it was decided
	// to add "real" transform, which uploads image as it is
	RealTransformName = "real"
	ImageExtension    = "jpg"
)

type imageData struct {
	Keys []string `json:"keys"`
	URL  string   `json:"url"`
}

type responseTemplate struct {
	Error   string      `json:"error"`
	Payload interface{} `json:"payload"`
}

type uploadResponsePayload struct {
	ImageKey        string            `json:"key"`
	OriginalURL     string            `json:"originalUrl"`
	Transformations map[string]string `json:"transformations"`
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

func makeTransformsPayload(imgKey string, transformationsURLs map[string]string) uploadResponsePayload {
	return uploadResponsePayload{
		ImageKey:        imgKey,
		OriginalURL:     transformationsURLs[OriginalTransformName],
		Transformations: transformationsURLs,
	}
}

func makePath(transformName, imageKey string) string {
	return fmt.Sprintf("%s/%s.%s", imageKey, transformName, ImageExtension)
}

func respondWithJSON(w http.ResponseWriter, err string, payload interface{}, code int) error {
	response := responseTemplate{Error: err, Payload: payload}
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
