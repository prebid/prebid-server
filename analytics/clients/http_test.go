package clients

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDefaultHttpInstance(t *testing.T) {
	assert.NotNil(t, GetDefaultHttpInstance())
}
