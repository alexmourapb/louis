package louis

import (
	"github.com/KazanExpress/louis/internal/pkg/utils"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"log"

	"github.com/KazanExpress/louis/internal/pkg/storage"
	redis2 "github.com/go-redis/redis"
)

const (
	RedisMaxActive = 5
	RedisMaxIdle   = 5
)

type AppContext struct {
	DB       *storage.DB
	Pool     *work.WorkerPool
	Config   *utils.Config
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
		appCtx.Pool.Drain()
		appCtx.Pool.Stop()
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
		MaxActive: RedisMaxActive,
		MaxIdle:   RedisMaxIdle,
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", appCtx.Config.RedisURL)
		},
	}

	appCtx.Pool = InitPool(appCtx, redisPool)
	appCtx.Enqueuer = work.NewEnqueuer(CleanupNamespace, redisPool)
	return appCtx
}
