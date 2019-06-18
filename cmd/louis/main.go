package main

import (
	"encoding/json"
	"github.com/KazanExpress/louis/internal/app/louis"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/KazanExpress/louis/internal/pkg/utils"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/procfs"
	"github.com/rs/cors"
	"io/ioutil"
	"log"
	"net/http"
	"runtime/debug"
	// _ "net/http/pprof"
	"gopkg.in/h2non/bimg.v1"
	"os"
	"time"
)

var (
	bimgMemory = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "bimg_memory_bytes",
		Help: "Amount of memory currently used by bimg",
	}, func() float64 {
		return float64(bimg.VipsMemory().Memory)
	})
)

func init() {
	prometheus.MustRegister(bimgMemory)
}

func addAccessControlAllowOriginHeader(cfg *utils.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", cfg.CORSAllowOrigin)
		w.Header().Add("Access-Control-Allow-Headers", cfg.CORSAllowHeaders)
		next.ServeHTTP(w, r)
	})
}

func getLouisAppContext() *louis.AppContext {
	var err error
	var appCtx = new(louis.AppContext)
	appCtx.Config = utils.InitConfig()
	appCtx.DB, err = storage.Open(appCtx.Config)

	if err != nil {
		log.Fatal(err)
	}

	if err = appCtx.DB.InitDB(); err != nil {
		log.Fatalf("FATAL: failed to init db - %v", err)
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

	// TODO: move cleanup to separate job (think about it)
	appCtx.WithWork()
	return appCtx
}

func runMemoryWatcher(appCtx *louis.AppContext) {
	if appCtx.Config.MemoryWatcherEnabled {
		var p, err = procfs.Self()
		if err != nil {
			log.Fatalf("ERROR: could not get process: %s", err)
		}

		stat, err := p.NewStat()
		if err != nil {
			log.Fatalf("ERROR: could not get process stat: %s", err)
		}

		var ticker = time.NewTicker(appCtx.Config.MemoryWatcherCheckInterval)
		go func() {
			for range ticker.C {
				var usedRezidentMemory = int64(stat.ResidentMemory())
				if usedRezidentMemory > appCtx.Config.MemoryWatcherLimitBytes {
					debug.FreeOSMemory()
				}
			}
		}()
	}
}

func main() {

	appCtx := getLouisAppContext()

	throttler := louis.NewThrottler(appCtx.Config)

	runMemoryWatcher(appCtx)

	// Register http handlers and start listening
	router := mux.NewRouter()
	router.HandleFunc("/", louis.GetDashboard).Methods("GET")
	router.Handle("/upload", throttler.Throttle(louis.UploadHandler(appCtx))).Methods("POST")
	router.Handle("/uploadWithClaim", throttler.Throttle(louis.UploadWithClaimHandler(appCtx))).Methods("POST")
	router.HandleFunc("/claim", louis.ClaimHandler(appCtx)).Methods("POST")
	router.HandleFunc("/healthz", louis.GetHealth(appCtx)).Methods("GET")

	utils.RegisterGracefulShutdown(func(signal os.Signal) {

		log.Printf("WARNING: Signal received: %s. Stoping...", signal.String())
		select {
		case <-time.After(appCtx.Config.GracefulShutdownTimeout):
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
	})
	crs := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},                      // All origins
		AllowedMethods: []string{"GET", "POST", "OPTIONS"}, // Allowing only get, just an example
	})
	go func() {
		var metricsRouter = mux.NewRouter()
		metricsRouter.Handle("/metrics", promhttp.Handler())
		metricsRouter.HandleFunc("/free", func(w http.ResponseWriter, req *http.Request) {
			debug.FreeOSMemory()
			w.WriteHeader(200)
		}).Methods("POST")
		log.Fatal(http.ListenAndServe(":8001", metricsRouter))
	}()

	// go func() {
	// 	// for pprof
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()
	log.Printf("INFO: app started!")
	log.Fatal(http.ListenAndServe(":8000", addAccessControlAllowOriginHeader(appCtx.Config, crs.Handler(router))))

}
