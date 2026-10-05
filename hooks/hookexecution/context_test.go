package hookexecution

import (
	"testing"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/stretchr/testify/assert"
)

func TestModuleContextsPutAfterNil(t *testing.T) {
	mcs := &moduleContexts{ctxs: map[string]*hookstage.ModuleContext{}}
	mcs.put("module", nil)

	later := hookstage.NewModuleContext()
	later.Set("key", "value")
	mcs.put("module", later)

	got, ok := mcs.get("module")
	assert.True(t, ok)
	v, found := got.Get("key")
	assert.True(t, found)
	assert.Equal(t, "value", v)
}

func TestModuleContextsPutMergesAndKeepsNilHarmless(t *testing.T) {
	mcs := &moduleContexts{ctxs: map[string]*hookstage.ModuleContext{}}

	first := hookstage.NewModuleContext()
	first.Set("a", 1)
	mcs.put("module", first)
	mcs.put("module", nil)

	second := hookstage.NewModuleContext()
	second.Set("b", 2)
	mcs.put("module", second)

	got, _ := mcs.get("module")
	assert.Equal(t, map[string]any{"a": 1, "b": 2}, got.GetAll())
}
