package main

import (
	"encoding/json"
	"flag"
	"github.com/KazanExpress/louis/internal/app/louis"
	"github.com/KazanExpress/louis/internal/pkg/queue"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
)

func addAcessControlAllowOriginHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}

func main() {
	envPath := flag.String("env", ".env", "path to file with environment variables")
	transformsPath := flag.String("transforms-path", "ensure-transforms.json", "path to file containing JSON transforms to ensure")
	initdb := flag.Bool("initdb", true, "if true then non-existing database tables will be created")
	flag.Parse()

	err := godotenv.Load(*envPath)
	if err != nil {
		log.Printf("INFO: failed to read env file: %v", err)
	}

	appCtx := &louis.AppContext{}
	appCtx.DB, err = storage.Open(os.Getenv("DATA_SOURCE_NAME"))
	if err != nil {
		log.Fatal(err)
	}

	if *initdb {
		if err = appCtx.DB.InitDB(); err != nil {
			log.Fatalf("FATAL: failed to init db - %v", err)
		}
	}

	if strings.ToLower(os.Getenv("TRANSFORMATIONS_ENABLED")) == "true" {
		log.Printf("INFO: TRANSFORMATIONS_ENABLED flag is set to TRUE")
		appCtx.TransformationsEnabled = true
		appCtx.Queue, err = queue.NewMachineryQueue(os.Getenv("REDIS_CONNECTION"))
		if err != nil {
			log.Fatalf("FATAL: failed to connect to redis instance - %v", err)
		}

		jsonBytes, err := ioutil.ReadFile(*transformsPath)
		if err != nil {
			log.Fatalf("FATAL: failed to read ensure-transforms.json - %v", err)
		}
		var tlist storage.TransformList
		err = json.Unmarshal(jsonBytes, &tlist)
		if err != nil {
			log.Fatalf("FATAL: failed to parse json from ensure-transforms.json - %v", err)
		}

		appCtx.DB.EnsureTransformations(tlist.Transformations)
	}

	// Register http handlers and start listening
	router := mux.NewRouter()
	router.HandleFunc("/", louis.GetDashboard).Methods("GET")
	router.Handle("/upload", louis.UploadHandler(appCtx)).Methods("POST")
	router.HandleFunc("/claim", louis.ClaimHandler(appCtx)).Methods("POST")
	router.HandleFunc("/healthz", louis.GetHealth(appCtx)).Methods("GET")
	// registering SIGTERM handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Printf("WARNING: Signal recieved: %s. Stoping...", sig.String())
			appCtx.DB.Lock()
			appCtx.DB.Close()

			os.Exit(2)
		}
	}()

	log.Fatal(http.ListenAndServe(":8000", addAcessControlAllowOriginHeader(handlers.CORS()((router)))))

}
