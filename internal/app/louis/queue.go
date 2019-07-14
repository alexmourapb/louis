package louis

import (
	// "github.com/KazanExpress/louis/internal/pkg/utils"
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

	err = appCtx.ImageService.Archive(imgKey)
	if err != nil {
		log.Printf("ERROR: failed to cleanup: %v", err)
		return err
	}

	log.Printf("CLEANUP_POOL: image with key=%v archived", imgKey)

	return nil
}

func InitPool(appCtx *AppContext, redisPool *redis.Pool) *work.WorkerPool {

	pool := work.NewWorkerPool(CleanupTaskCtx{}, appCtx.Config.CleanupPoolConcurrency, CleanupNamespace, redisPool)

	pool.Job(CleanupTask, (*CleanupTaskCtx).Cleanup)

	pool.Middleware(func(c *CleanupTaskCtx, job *work.Job, next work.NextMiddlewareFunc) error {
		c.AppContext = appCtx
		return next()
	})

	pool.Start()

	return pool
}
