package louis

import (
	"github.com/KazanExpress/louis/internal/pkg/config"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestThrottlerLock(t *testing.T) {
	config := config.InitFrom("../../../.env")
	config.ThrottlerTimeout = 5 * time.Second
	throt := NewThrottler(config)

	assert := assert.New(t)

	for i := int64(0); i < config.ThrottlerQueueLength; i++ {
		assert.True(throt.Lock())
	}

	assert.False(throt.Lock())
}

func TestThrottlerUnlock(t *testing.T) {
	config := config.InitFrom("../../../.env")
	config.ThrottlerTimeout = 5 * time.Second
	throt := NewThrottler(config)

	assert := assert.New(t)

	for i := int64(0); i < config.ThrottlerQueueLength; i++ {
		assert.True(throt.Lock())
	}

	throt.Unlock()

	assert.True(throt.Lock())
}
