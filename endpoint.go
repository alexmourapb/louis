package main

import (
	"net/http"
	"fmt"
	"log"
	"io"
	"bytes"
	"io/ioutil"
	"time"
	"strconv"
)

func GetDashboard(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, here is your dashboard")
}

func Upload(w http.ResponseWriter, r *http.Request) {
	var t = time.Now()
	r.ParseMultipartForm(5 * 1024 * 1024)
	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	fmt.Fprintf(w, "%v", handler.Header)
	// TODO: change the following lines to work with S3 storage
	var buffer bytes.Buffer
	if err != nil {
		log.Println(err)
		return
	}
	io.Copy(&buffer, file)
	err = ioutil.WriteFile("./docs/images/" + strconv.Itoa(int(t.Unix())) + ".jpg", buffer.Bytes(), 0644)
	if err != nil {
		log.Println(err)
		return
	}
}