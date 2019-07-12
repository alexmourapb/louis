package main

import (
	"encoding/json"
	"github.com/KazanExpress/louis/internal/app/louis"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/KazanExpress/louis/internal/pkg/utils"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
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

func getLouisAppContext() *louis.AppContext {
	var err error
	var appCtx = new(louis.AppContext)
	appCtx.Config = utils.InitConfig()
	appCtx.DB, err = storage.Open(appCtx.Config)
	appCtx.ImageService = louis.NewLouisService(appCtx)

	if err != nil {
		log.Fatal(err)
	}

	appCtx.Storage, err = storage.InitS3Context(appCtx.Config)
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

	runMemoryWatcher(appCtx)

	// Register http handlers and start listening
	var server = louis.NewServer(appCtx)

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

	go func() {
		log.Fatal(http.ListenAndServe(":8001", server.MetricsRouter()))
	}()

	// go func() {
	// 	// for pprof
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()
	log.Printf("INFO: app started!")
	log.Fatal(http.ListenAndServe(":8000", server.AppRouter()))
}
