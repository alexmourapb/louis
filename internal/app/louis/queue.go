package louis

import (
	"github.com/KazanExpress/louis/internal/pkg/storage"
	// "github.com/KazanExpress/louis/internal/pkg/config"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"log"
)

const (
	CleanupNamespace = "cleanup_pool_namespace"
	CleanupTask      = "delete_images"
)

type CleanupTaskCtx struct {
	*AppContext
	ImageKey string
}

func (appCtx *CleanupTaskCtx) Cleanup(job *work.Job) error {
	appCtx.AppContext = GetGlobalCtx()
	log.Printf("CLEANUP_POOL: recieved task with args [%v]", job.Args)

	var imgKey = job.ArgString("key")

	log.Printf("here1")
	log.Printf("%v", appCtx)
	img, err := appCtx.DB.QueryImageByKey(imgKey)
	if err != nil {
		return err
	}
	log.Printf("here2")
	if img.Approved {
		log.Printf("INFO: image with key=%v is approved, nothing to delete", imgKey)
		return nil
	}
	log.Printf("INFO: image with key=%v is not approved, deleting it", imgKey)
	err = storage.DeleteFolder(imgKey)
	log.Printf("here3")
	if err != nil {
		return err
	}
	defer log.Printf("INFO: image with key=%v deletd", imgKey)
	return appCtx.DB.DeleteImage(imgKey)
}

func InitPool(appCtx *AppContext, redisPool *redis.Pool) *work.WorkerPool {

	log.Printf("ms: %v", appCtx)
	SetGlobalCtx(appCtx)

	pool := work.NewWorkerPool(CleanupTaskCtx{}, appCtx.Config.CleanupPoolConcurrency, CleanupNamespace, redisPool)
	// pool.Middleware((*CleanupTaskCtx).Log)

	pool.Job(CleanupTask, (*CleanupTaskCtx).Cleanup)

	pool.Start()

	return pool
}
