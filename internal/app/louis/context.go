package louis

import (
	"github.com/KazanExpress/louis/internal/pkg/config"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"log"

	"github.com/KazanExpress/louis/internal/pkg/storage"
	redis2 "github.com/go-redis/redis"
)

type AppContext struct {
	DB       *storage.DB
	Pool     *work.WorkerPool
	Config   *config.Config
	Enqueuer *work.Enqueuer
	Dropped  bool
}

func SetGlobalCtx(ctx *AppContext) {
	globalCtx = ctx
}

func GetGlobalCtx() *AppContext {
	return globalCtx
}

var globalCtx *AppContext

func (appCtx *AppContext) DropAll() error {

	if appCtx == nil {
		return nil
	}

	if appCtx.Dropped {
		return nil
	}

	appCtx.Dropped = true

	if appCtx.Pool != nil {
		log.Printf("POOL: draining >")
		appCtx.Pool.Drain()
		log.Printf("POOL: drained =>")
		log.Printf("POOL: stoping >")
		appCtx.Pool.Stop()
		log.Printf("POOL: stoped=>")
	}

	appCtx.DropRedis()
	return appCtx.DB.DropDB()
}

func (appCtx *AppContext) DropRedis() {
	client := redis2.NewClient(&redis2.Options{
		Addr:     appCtx.Config.RedisURL,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	err := client.FlushAll().Err()
	if err != nil {
		log.Printf("WARN: failed to drop redis - %v", err)
	}
}

func (appCtx *AppContext) WithWork() *AppContext {
	var redisPool = &redis.Pool{
		MaxActive: 5,
		MaxIdle:   5,
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", appCtx.Config.RedisURL)
		},
	}

	appCtx.Pool = InitPool(appCtx, redisPool)
	appCtx.Enqueuer = work.NewEnqueuer(CleanupNamespace, redisPool)
	return appCtx
}
