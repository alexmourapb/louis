package main

import (
	"encoding/json"
	"github.com/KazanExpress/louis/internal/app/louis"
	"github.com/KazanExpress/louis/internal/pkg/config"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func addAccessControlAllowOriginHeader(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", cfg.CORSAllowOrigin)
		w.Header().Add("Access-Control-Allow-Headers", cfg.CORSAllowHeaders)
		next.ServeHTTP(w, r)
	})
}

func initApp(appCtx *louis.AppContext) {
	var err error
	appCtx.Config = config.Init()
	appCtx.DB, err = storage.Open(appCtx.Config)
	if err != nil {
		log.Fatal(err)
	}

	if appCtx.Config.InitDB {
		if err = appCtx.DB.InitDB(); err != nil {
			log.Fatalf("FATAL: failed to init db - %v", err)
		}
	}

	jsonBytes, err := ioutil.ReadFile(appCtx.Config.TransformsPath)
	if err != nil {
		log.Fatalf("FATAL: failed to read ensure-transforms.json - %v", err)
	}
	var tlist storage.TransformList
	err = json.Unmarshal(jsonBytes, &tlist)
	if err != nil {
		log.Fatalf("FATAL: failed to parse json from ensure-transforms.json - %v", err)
	}

	err = appCtx.DB.EnsureTransformations(tlist.Transformations)
	if err != nil {
		log.Printf("ERROR: failed to ensure transformations: %v", err)
	}

	appCtx.WithWork()
}

func main() {

	appCtx := &louis.AppContext{}
	initApp(appCtx)

	// Register http handlers and start listening
	router := mux.NewRouter()
	router.HandleFunc("/", louis.GetDashboard).Methods("GET")
	router.Handle("/upload", louis.UploadHandler(appCtx)).Methods("POST")
	router.Handle("/uploadWithClaim", louis.UploadWithClaimHandler(appCtx)).Methods("POST")
	router.HandleFunc("/claim", louis.ClaimHandler(appCtx)).Methods("POST")
	router.HandleFunc("/healthz", louis.GetHealth(appCtx)).Methods("GET")
	// registering SIGTERM handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Printf("WARNING: Signal recieved: %s. Stoping...", sig.String())
			select {
			case <-time.After(time.Second * 10):
				appCtx.Pool.Stop()
				break
			case <-func() chan bool {
				ch := make(chan bool, 1)
				ch <- true
				appCtx.Pool.Drain()
				return ch
			}():
				log.Printf("INFO: worker pool drained successfully")
				break
			}
			os.Exit(2)

		}
	}()

	crs := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},                      // All origins
		AllowedMethods: []string{"GET", "POST", "OPTIONS"}, // Allowing only get, just an example
	})
	log.Printf("INFO: app started!")

	log.Fatal(http.ListenAndServe(":8000", addAccessControlAllowOriginHeader(appCtx.Config, crs.Handler(router))))

}
