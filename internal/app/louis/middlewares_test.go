package louis

import (
	"github.com/KazanExpress/louis/internal/pkg/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestThrottlerLock(t *testing.T) {
	config := utils.InitConfigFrom("../../../.env")
	config.ThrottlerTimeout = 5 * time.Second
	throt := NewThrottler(config)

	assert := assert.New(t)

	for i := int64(0); i < config.ThrottlerQueueLength; i++ {
		assert.True(throt.lock())
	}

	assert.False(throt.lock())
}

func TestThrottleuUnlock(t *testing.T) {
	config := utils.InitConfigFrom("../../../.env")
	config.ThrottlerTimeout = 5 * time.Second
	throt := NewThrottler(config)

	assert := assert.New(t)

	for i := int64(0); i < config.ThrottlerQueueLength; i++ {
		assert.True(throt.lock())
	}

	throt.unlock()

	assert.True(throt.lock())
}
