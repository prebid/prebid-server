//go:build !cgo

package vastlint

import (
	"testing"

	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/stretchr/testify/require"
)

func TestBuilderRequiresCGO(t *testing.T) {
	_, err := Builder([]byte(`{"enabled":true}`), moduledeps.ModuleDeps{})
	require.EqualError(t, err, "openadtech.vastlint requires cgo")
}
