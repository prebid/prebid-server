package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLayerToken(t *testing.T) {
	assert.Equal(t, "l1", LayerL1.Token())
	assert.Equal(t, "l2", LayerL2.Token())
	assert.Equal(t, "none", LayerNone.Token(), "a full miss reached no layer")
}

func TestToResult(t *testing.T) {
	assert.Equal(t, InProgress, toResult(Entry{InProgress: true}, FirstParty, LayerL1).State)
	assert.Equal(t, Negative, toResult(Entry{Negative: true}, FirstParty, LayerL1).State)

	hit := toResult(Entry{}, ThirdParty, LayerL2)
	assert.Equal(t, Hit, hit.State)
	assert.Equal(t, ThirdParty, hit.KeyType)
	assert.Equal(t, LayerL2, hit.Layer)
}
