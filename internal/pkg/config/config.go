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
	CleanupPoolConcurrency uint   `envconfig:"CLEANUP_POOL_CONCURRENCY" default:"10"`
	// In minutes
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

	err = envconfig.Process("louis", App)
	if err != nil {
		panic(err)
	}

	return App
}
