package louis

import (
	"net/http"
	"fmt"
	"log"
	"io"
	"bytes"
	"time"
	"strconv"
	"encoding/json"
	"github.com/KazanExpress/Louis/internal/pkg/utils"
)

type ImageKey struct {
	Key string `json:"key"`
	Url string `json:"url"`
}

func GetDashboard(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, here is your dashboard")
}

func Upload(w http.ResponseWriter, r *http.Request) {
	var t = time.Now()
	r.ParseMultipartForm(5 * 1024 * 1024)
	file, _, err := r.FormFile("file")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	var buffer bytes.Buffer
	if err != nil {
		log.Println(err)
		return
	}
	io.Copy(&buffer, file)
	var imageKey ImageKey
	imageKey.Key = strconv.Itoa(int(t.Unix()))
	output, err := utils.UploadFile(bytes.NewReader(buffer.Bytes()), imageKey.Key + ".jpg")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	imageKey.Url = output.Location
	js, err := json.Marshal(imageKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}