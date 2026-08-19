package cache

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
