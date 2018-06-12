package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"github.com/gorilla/mux"
	"net/http"
)

func main() {
	router := mux.NewRouter();
	router.HandleFunc("/", GetDashboard).Methods("GET")
	router.HandleFunc("/upload", Upload).Methods("POST")
	log.Fatal(http.ListenAndServe(":8000", router))

	err := godotenv.Load()
	if err != nil {
		log.Printf("INFO: .env file not found using real env variables")
	}

	s3Bucket := os.Getenv("S3_BUCKET")
	secretKey := os.Getenv("SECRET_KEY")

	fmt.Printf("%s %s", s3Bucket, secretKey)
	// now do something with s3 or whatever
}
