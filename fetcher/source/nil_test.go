package source

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNilSourceReturnsNotFound(t *testing.T) {
	raw, found, err := (NilSource[string]{}).Fetch(context.Background(), "missing")

	require.NoError(t, err)
	assert.False(t, found)
	assert.Nil(t, raw)
}
