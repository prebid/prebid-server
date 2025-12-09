package file

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/analyticsdeps"
	"github.com/stretchr/testify/assert"
)

func TestBuilderEmptyConfig(t *testing.T) {
	mod, err := Builder(nil, analyticsdeps.Deps{})
	assert.NoError(t, err)
	assert.Nil(t, mod)

	mod, err = Builder([]byte(`{}`), analyticsdeps.Deps{})
	assert.NoError(t, err)
	assert.Nil(t, mod)
}

func TestBuilderInvalidConfig(t *testing.T) {
	mod, err := Builder([]byte(`{`), analyticsdeps.Deps{})
	assert.Error(t, err)
	assert.Nil(t, mod)
}

func TestBuilderEnabled(t *testing.T) {
	cfg := Config{
		Filename: "test-file.log",
	}
	raw, err := json.Marshal(cfg)
	assert.NoError(t, err)

	mod, err := Builder(raw, analyticsdeps.Deps{})
	assert.NoError(t, err)
	assert.NotNil(t, mod)

	_, ok := mod.(analytics.Module)
	assert.True(t, ok)
}
