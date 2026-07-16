package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyTypeToken(t *testing.T) {
	assert.Equal(t, "first_party", FirstParty.Token())
	assert.Equal(t, "third_party", ThirdParty.Token())
	assert.Equal(t, "device", Device.Token())
	assert.Equal(t, "unknown", KeyType(99).Token())
}

func TestLayerToken(t *testing.T) {
	assert.Equal(t, "l1", LayerL1.Token())
	assert.Equal(t, "l2", LayerL2.Token())
	assert.Equal(t, "unknown", LayerNone.Token())
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

func TestDecodeValid(t *testing.T) {
	future := time.Now().Add(time.Hour).UnixMilli()

	assert.Nil(t, decodeValid(nil), "empty -> nil")
	assert.Nil(t, decodeValid([]byte(`{bad`)), "invalid JSON -> nil")

	past, _ := json.Marshal(Entry{Exp: time.Now().Add(-time.Hour).UnixMilli()})
	assert.Nil(t, decodeValid(past), "expired -> nil")

	live, _ := json.Marshal(Entry{Negative: true, Exp: future})
	got := decodeValid(live)
	require.NotNil(t, got)
	assert.True(t, got.Negative)
}

func TestToResult(t *testing.T) {
	assert.Equal(t, InProgress, toResult(Entry{InProgress: true}, FirstParty, LayerL1).State)
	assert.Equal(t, Negative, toResult(Entry{Negative: true}, FirstParty, LayerL1).State)

	hit := toResult(Entry{}, ThirdParty, LayerL2)
	assert.Equal(t, Hit, hit.State)
	assert.Equal(t, ThirdParty, hit.KeyType)
	assert.Equal(t, LayerL2, hit.Layer)
}

func TestEvictedKeysError(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewRedisStore(client)
	mr.Close() // force the INFO call to fail

	_, err := store.EvictedKeys(context.Background())
	assert.Error(t, err)
}
