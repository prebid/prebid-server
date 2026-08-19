package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider/memory"
)

func TestSetErrFailsGetAndPut(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	require.NoError(t, store.Put(ctx, "k", "v", time.Minute))

	down := errors.New("backend down")
	store.SetErr(down)
	_, err := store.Get(ctx, "k")
	assert.ErrorIs(t, err, down)
	assert.ErrorIs(t, store.Put(ctx, "k", "v2", time.Minute), down)

	store.SetErr(nil)
	value, err := store.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "v", value)
}

func TestTTLAndLen(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	assert.Equal(t, time.Duration(0), store.TTL("absent"))

	require.NoError(t, store.Put(ctx, "k", "v", time.Minute))
	assert.InDelta(t, time.Minute, store.TTL("k"), float64(2*time.Second))
	assert.Equal(t, 1, store.Len())

	store.Advance(2 * time.Minute)
	assert.Equal(t, time.Duration(0), store.TTL("k"), "an expired entry reports no remaining TTL")
	assert.Equal(t, 0, store.Len())
}
