package louis

import (
	"bytes"
	"context"
	"fmt"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/KazanExpress/louis/internal/pkg/utils"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"golang.org/x/sync/semaphore"
	"image"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// NewThrottler - contsructor for Throttler
func NewThrottler(cfg *utils.Config) *Throttler {
	return &Throttler{
		semaphore: semaphore.NewWeighted(cfg.ThrottlerQueueLength),
		timeout:   cfg.ThrottlerTimeout,
	}
}

// Throttler - simple middleware to throttle traffic
type Throttler struct {
	semaphore *semaphore.Weighted
	timeout   time.Duration
}

// Lock - tries to acquire right to handle request
func (t *Throttler) lock() bool {
	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()
	var err = t.semaphore.Acquire(ctx, 1)
	return err == nil
}

func (t *Throttler) unlock() {
	t.semaphore.Release(1)
}

// Throttle - locks until request can be handled or timeout
func (t *Throttler) Throttle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if t.lock() {
			defer t.unlock()
			next.ServeHTTP(w, r)
		} else {
			err := respondWithJSON(w, "too many requests", nil, 503)
			if err != nil {
				log.Printf("ERROR: failed to respond with 'too many requests' - %s", err)
				w.WriteHeader(503)
			}
		}
	})
}

func recoverFromPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("ERROR: catch panic %v", err)
				respondWithJSON(w, "internal server error", nil, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func addAccessControlAllowOriginHeader(cfg *utils.Config) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Access-Control-Allow-Origin", cfg.CORSAllowOrigin)
			w.Header().Add("Access-Control-Allow-Headers", cfg.CORSAllowHeaders)
			next.ServeHTTP(w, r)
		})
	}
}

func corsMiddleware() mux.MiddlewareFunc {
	var crs = cors.New(cors.Options{
		AllowedOrigins: []string{"*"},                      // All origins
		AllowedMethods: []string{"GET", "POST", "OPTIONS"}, // Allowing only get, just an example
	})
	return crs.Handler
}

// Authorize - creates authorization middleware by given key
func authorize(key string) func(sessionHandler) sessionHandler {
	return func(next sessionHandler) sessionHandler {
		return sessionHandler(func(s *session, w http.ResponseWriter, r *http.Request) {
			var header = r.Header.Get("Authorization")
			if header != key {
				respondWithJSON(w, "account not found", nil, http.StatusUnauthorized)
				return
			}
			s.userID = 1
			next(s, w, r)
		})
	}
}

func validate() func(sessionHandler) sessionHandler {

	return func(next sessionHandler) sessionHandler {
		return sessionHandler(func(s *session, w http.ResponseWriter, r *http.Request) {
			s.args = new(requestArgs)

			if r.ContentLength > s.ctx.Config.MaxImageSize {
				respondWithJSON(w, fmt.Sprintf("image size should be less than  %v bytes", s.ctx.Config.MaxImageSize), nil, http.StatusBadRequest)
				return
			}

			var err = r.ParseMultipartForm(s.ctx.Config.MaxImageSize)
			if failOnError(w, err, "error on parsing multipart form", http.StatusBadRequest) {
				return
			}

			file, _, err := r.FormFile("file")

			if failOnError(w, err, "error on reading file from multipart", http.StatusBadRequest) {
				return
			}

			defer file.Close()

			var tagsStr = strings.Replace(r.FormValue("tags"), " ", "", -1)
			if tagsStr != "" {

				s.args.tags = strings.Split(tagsStr, ",")
				for _, tag := range s.args.tags {
					if len(tag) > storage.TagLength {
						respondWithJSON(w, fmt.Sprintf("tag should not be longer than %v", storage.TagLength), nil, http.StatusBadRequest)
						return
					}
				}
			}

			var buffer bytes.Buffer
			io.Copy(&buffer, file)
			s.args.image = buffer.Bytes()

			_, _, err = image.Decode(bytes.NewReader(s.args.image))
			if failOnError(w, err, "error on creating an Image object from bytes", http.StatusBadRequest) {
				return
			}

			keyArg := r.FormValue("key")
			if keyArg != "" {
				s.args.imageKey = keyArg
			}

			next(s, w, r)
		})
	}
}
