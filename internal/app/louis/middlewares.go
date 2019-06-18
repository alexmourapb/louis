package louis

import (
	"context"
	"github.com/KazanExpress/louis/internal/pkg/utils"
	"golang.org/x/sync/semaphore"
	"log"
	"net/http"
	"time"
)

func NewThrottler(cfg *utils.Config) *Throttler {
	return &Throttler{
		semaphore: semaphore.NewWeighted(cfg.ThrottlerQueueLength),
		timeout:   cfg.ThrottlerTimeout,
	}
}

type Throttler struct {
	semaphore *semaphore.Weighted
	timeout   time.Duration
}

// Lock - tries to acquire right to handle request
func (t *Throttler) Lock() bool {
	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()
	var err = t.semaphore.Acquire(ctx, 1)
	return err == nil
}

func (t *Throttler) Unlock() {
	t.semaphore.Release(1)
}

func (t *Throttler) Throttle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if t.Lock() {
			defer t.Unlock()
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
