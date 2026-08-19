package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigTTLPolicy(t *testing.T) {
	c := Config{
		TTLSeconds:                  100,
		TTLCeilingFirstPartySeconds: 200,
		TTLCeilingThirdPartySeconds: 300,
		TTLCeilingDeviceSeconds:     400,
		NegativeTTLSeconds:          5,
		InProgressTTLSeconds:        6,
	}

	p := c.TTLPolicy()
	assert.Equal(t, 100*time.Second, p.Default)
	assert.Equal(t, 200*time.Second, p.FirstPartyCeiling)
	assert.Equal(t, 300*time.Second, p.ThirdPartyCeiling)
	assert.Equal(t, 400*time.Second, p.DeviceCeiling)
	assert.Equal(t, 5*time.Second, p.NegativeTTL)
	assert.Equal(t, 6*time.Second, p.InProgressTTL)
}
