package main

import (
	// "fmt"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", GetDashboard).Methods("GET")
	router.HandleFunc("/upload", Upload).Methods("POST")
	log.Fatal(http.ListenAndServe(":8000", router))

	err := godotenv.Load()
	if err != nil {
		log.Printf("INFO: .env file not found using real env variables")
	}

	// testS3()
	// now do something with s3 or whatever
}
