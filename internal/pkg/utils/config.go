package utils

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/namsral/flag"
	"log"
	"time"
)

// Config - application configs
type Config struct {
	RedisURL               string `envconfig:"REDIS_URL" default:":6379"`
	TransformsPath         string `ignored:"true"`
	CleanupPoolConcurrency uint   `envconfig:"CLEANUP_POOL_CONCURRENCY" default:"10"`
	// In minutes; TODO: -> 1m
	CleanUpDelay int `envconfig:"CLEANUP_DELAY" default:"1"`

	PostgresUser     string `envconfig:"POSTGRES_USER" default:"postgres"`
	PostgresPassword string `envconfig:"POSTGRES_PASSWORD" default:""`
	PostgresAddress  string `envconfig:"POSTGRES_ADDRESS" default:"127.0.0.1:5432"`
	PostgresDatabase string `envconfig:"POSTGRES_DATABASE" default:"postgres"`
	PostgresSSLMode  string `envconfig:"POSTGRES_SSL_MODE" default:"disable"`

	CORSAllowOrigin  string `envconfig:"CORS_ALLOW_ORIGIN" default:"*"`
	CORSAllowHeaders string `envconfig:"CORS_ALLOW_HEADERS" default:"Authorization,Content-Type,Access-Content-Allow-Origin"`
	// MaxImageSize maximum image size in bytes, default is 5MB
	MaxImageSize int64 `envconfig:"MAX_IMAGE_SIZE" default:"5242880"`

	ThrottlerQueueLength int64  `envconfig:"THROTTLER_QUEUE_LENGTH" default:"10"`
	ThrottlerTimeoutStr  string `envconfig:"THROTTLER_TIMEOUT" default:"15s"`
	ThrottlerTimeout     time.Duration

	MemoryWatcherEnabled       bool          `envconfig:"MEMORY_WATCHER_ENABLED" default:"false"`
	MemoryWatcherLimitBytes    int64         `envconfig:"MEMORY_WATCHER_LIMIT_BYTES" default:"1610612736"` // 1.5 GB
	MemoryWatcherCheckInterval time.Duration `envconfig:"MEMORY_WATCHER_CHECK_INTERVAL" default:"10m"`     // 1.5 GB

	GracefulShutdownTimeout time.Duration `default:"10s" split_words:"true"`
}

// App - application configs

func InitConfig() *Config {

	envPath := flag.String("env", ".env", "path to file with environment variables")
	transformsPath := flag.String("transforms-path", "ensure-transforms.json", "path to file containing JSON transforms to ensure")

	flag.Parse()

	conf := InitConfigFrom(*envPath)
	conf.TransformsPath = *transformsPath
	return conf
}

// InitConfigFrom - initializes configs from env file
func InitConfigFrom(envPath string) *Config {

	App := &Config{}

	err := godotenv.Load(envPath)
	if err != nil {
		log.Printf("INFO: failed to read env file: %v", err)
	}

	err = envconfig.Process("louis", App)

	if err != nil {
		panic(err)
	}

	App.ThrottlerTimeout, err = time.ParseDuration(App.ThrottlerTimeoutStr)

	if err != nil {
		panic(err)
	}

	return App
}
