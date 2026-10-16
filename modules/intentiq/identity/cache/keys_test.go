package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestKeyTypeToken(t *testing.T) {
	assert.Equal(t, "first_party", FirstParty.Token())
	assert.Equal(t, "third_party", ThirdParty.Token())
	assert.Equal(t, "device", Device.Token())
	assert.Equal(t, "unknown", KeyType(99).Token())
}

func TestCeilingFor(t *testing.T) {
	p := TTLPolicy{
		Default:           1 * time.Second,
		FirstPartyCeiling: 2 * time.Second,
		ThirdPartyCeiling: 3 * time.Second,
		DeviceCeiling:     4 * time.Second,
	}
	assert.Equal(t, 2*time.Second, p.CeilingFor(FirstParty))
	assert.Equal(t, 3*time.Second, p.CeilingFor(ThirdParty))
	assert.Equal(t, 4*time.Second, p.CeilingFor(Device))
	assert.Equal(t, 1*time.Second, p.CeilingFor(KeyType(99)), "unknown type falls back to Default")
}
