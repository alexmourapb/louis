package main

import (
	"flag"
	"github.com/KazanExpress/Louis/internal/app/louis"
	"github.com/KazanExpress/Louis/internal/pkg/storage"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/streadway/amqp"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("INFO: .env file not found using real env variables")
	}

	appCtx := &louis.AppContext{}

	if strings.ToLower(os.Getenv("TRANSFORMATIONS_ENABLED")) == "true" {
		log.Printf("INFO: TRANSFORMATIONS_ENABLED flag is set to TRUE")
		appCtx.RabbitMQConnection, err = amqp.Dial(os.Getenv("RABBITMQ_CONNECTION"))
		if err != nil {
			log.Fatalf("ERROR: failed to connect to RabbitMQ instance - %v", err)
		}
	}

	appCtx.DB, err = storage.Open(os.Getenv("DATA_SOURCE_NAME"))
	initdb := flag.Bool("initdb", false, "if true then non-existing database tables will be created")
	flag.Parse()

	if *initdb {
		if err = appCtx.DB.InitDB(); err != nil {
			log.Fatalf("ERROR: failed to init db - %v", err)
		}
	}

	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/", louis.GetDashboard).Methods("GET")
	router.Handle("/upload", louis.UploadHandler(appCtx)).Methods("POST")
	router.HandleFunc("/claim", louis.ClaimHandler(appCtx)).Methods("POST")
	log.Fatal(http.ListenAndServe(":8000", router))

	// testS3()
	// now do something with s3 or whatever
}
