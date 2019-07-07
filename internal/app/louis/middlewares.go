package louis

import (
	"context"
	"github.com/KazanExpress/louis/internal/pkg/utils"
	"golang.org/x/sync/semaphore"
	"log"
	"net/http"
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

// Authorize - creates authorization middleware by given key
func Authorize(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var header = r.Header.Get("Authorization")
			if header != key {
				respondWithJSON(w, "account not found", nil, http.StatusUnauthorized)
			}
		})
	}
}
