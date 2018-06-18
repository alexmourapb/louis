package main

import (
	"github.com/KazanExpress/Louis/internal/app/louis"
	"github.com/KazanExpress/Louis/internal/pkg/storage"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"os"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("INFO: .env file not found using real env variables")
	}

	database, err := storage.Open(os.Getenv("DATA_SOURCE_NAME"))
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/", louis.GetDashboard).Methods("GET")
	router.Handle("/upload", louis.UploadHandler(database)).Methods("POST")
	router.HandleFunc("/claim", louis.ClaimHandler(database)).Methods("POST")
	log.Fatal(http.ListenAndServe(":8000", router))

	// testS3()
	// now do something with s3 or whatever
}
