package tmp

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/stretchr/testify/require"
)

func TestBuilder_EmptyConfig(t *testing.T) {
	m, err := Builder(json.RawMessage(`{}`), moduledeps.ModuleDeps{})
	require.NoError(t, err)
	require.NotNil(t, m)
}
