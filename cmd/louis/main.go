package main

import (
	// "fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"github.com/KazanExpress/Louis/internal/app/louis"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("INFO: .env file not found using real env variables")
	}

	router := mux.NewRouter()
	router.HandleFunc("/", louis.GetDashboard).Methods("GET")
	router.HandleFunc("/upload", louis.Upload).Methods("POST")
	router.HandleFunc("/claim", louis.Claim).Methods("POST")
	log.Fatal(http.ListenAndServe(":8000", router))

	// testS3()
	// now do something with s3 or whatever
}
