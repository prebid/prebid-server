//go:build !wurfl

package wurfl_devicedetection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWurflEngineFallbackError(t *testing.T) {
	_, err := newWurflEngine(config{})
	assert.EqualError(t, err, wurflBuildTagMissingError)
}
