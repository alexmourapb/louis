package main

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"github.com/KazanExpress/Louis/internal/app/louis"
	"github.com/KazanExpress/Louis/internal/pkg/db"
	"github.com/joho/godotenv"
	"os"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("INFO: .env file not found using real env variables")
	}

	database, err := db.Open(os.Getenv("DATA_SOURCE_NAME"))
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/", louis.GetDashboard).Methods("GET")
	router.Handle("/upload", louis.UploadHandler(database)).Methods("POST")
	router.HandleFunc("/claim", louis.Claim).Methods("POST")
	log.Fatal(http.ListenAndServe(":8000", router))

	// testS3()
	// now do something with s3 or whatever
}
