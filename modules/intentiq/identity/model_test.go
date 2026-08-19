package identity

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

func TestFlowContextRoundTrip(t *testing.T) {
	fc := flowContext{abTestUUID: "ab", auctionID: "auc"}
	got, ok := getFlowContext(setFlowContext(fc))
	assert.True(t, ok)
	assert.Equal(t, fc, got)
}

func TestGetFlowContextMissing(t *testing.T) {
	_, ok := getFlowContext(nil)
	assert.False(t, ok, "nil module context -> not present")

	_, ok = getFlowContext(hookstage.NewModuleContext())
	assert.False(t, ok, "empty module context -> not present")

	mctx := hookstage.NewModuleContext()
	mctx.Set(flowContextKey, "not a flowContext")
	_, ok = getFlowContext(mctx)
	assert.False(t, ok, "wrong type under key -> not present")
}
