package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/namsral/flag"
	"log"
)

// Config - application configs
type Config struct {
	RedisURL               string `envconfig:"REDIS_URL" default:":6379"`
	TransformsPath         string `ignored:"true"`
	InitDB                 bool   `ignored:"true"`
	DataSourceName         string `envconfig:"DATA_SOURCE_NAME" default:"db.sqlite"`
	CleanupPoolConcurrency uint   `envconfig:"CLEANUP_POOL_CONCURRENCY" default:"10"`
	// In minutes
	CleanUpDelay int `envconfig:"CLEANUP_DELAY" default:"1"`
}

// App - application configs

func Init() *Config {

	envPath := flag.String("env", ".env", "path to file with environment variables")
	transformsPath := flag.String("transforms-path", "ensure-transforms.json", "path to file containing JSON transforms to ensure")
	initDB := flag.Bool("initdb", true, "if true then non-existing database tables will be created")

	flag.Parse()

	conf := InitFrom(*envPath)
	conf.InitDB = *initDB
	conf.TransformsPath = *transformsPath
	return conf
}

// InitFrom - initializes configs from env file
func InitFrom(envPath string) *Config {

	App := &Config{}

	err := godotenv.Load(envPath)
	if err != nil {
		log.Printf("INFO: failed to read env file: %v", err)
	}

	err = envconfig.Process("myapp", App)
	if err != nil {
		panic(err)
	}
	// flag.StringVar(&App.RedisURL, "redis-url", ":6379", "adress of redis instance")
	// flag.StringVar(&App.DataSourceName, "data-source-name", "db.sqlite", "database uri string")
	// flag.UintVar(&App.CleanupPoolConcurrency, "cleanup-pool-concurrency", 10, "max number of concurrent tasks cleaning up trash images")
	// flag.IntVar(&App.CleanUpDelay, "cleanup-delay", 1, "after this delay unclaimed images will be deleted")

	return App
}
