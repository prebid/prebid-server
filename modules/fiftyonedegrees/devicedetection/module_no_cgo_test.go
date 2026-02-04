//go:build !cgo

package devicedetection

import (
	"testing"

	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/stretchr/testify/assert"
)

func TestBuilderError(t *testing.T) {
	_, err := Builder(nil, moduledeps.ModuleDeps{})
	assert.EqualError(t, err, errMsg)
}
