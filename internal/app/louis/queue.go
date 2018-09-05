package louis

import (
	"github.com/KazanExpress/louis/internal/pkg/storage"
	// "github.com/KazanExpress/louis/internal/pkg/config"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	_ "github.com/mattn/go-sqlite3"

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
	log.Printf("CLEANUP_POOL: received task with args [%v]", job.Args)

	var imgKey = job.ArgString("key")

	img, err := appCtx.DB.QueryImageByKey(imgKey)
	if err != nil {
		return err
	}
	if img.Approved {
		log.Printf("CLEANUP_POOL: image with key=%v is approved, nothing to delete", imgKey)
		return nil
	}
	log.Printf("CLEANUP_POOL: image with key=%v is not approved, deleting it", imgKey)
	err = storage.DeleteFolder(imgKey)
	if err != nil {
		return err
	}
	defer log.Printf("CLEANUP_POOL: image with key=%v deleted", imgKey)
	return appCtx.DB.DeleteImage(imgKey)
}

func InitPool(appCtx *AppContext, redisPool *redis.Pool) *work.WorkerPool {

	SetGlobalCtx(appCtx)

	pool := work.NewWorkerPool(CleanupTaskCtx{}, appCtx.Config.CleanupPoolConcurrency, CleanupNamespace, redisPool)

	pool.Job(CleanupTask, (*CleanupTaskCtx).Cleanup)

	pool.Start()

	return pool
}
