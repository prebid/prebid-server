package optimizationmodule

import (
	"encoding/json"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExecuteRules(t *testing.T) {

	rules := Build(json.RawMessage{})
	rw := &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			Device: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Country: "USA",
				},
				IP: "0.0.0.0",
			},
		}}
	err := rules.Execute(rw)
	assert.Equal(t, "127.0.0.1", rw.Device.IP)
	assert.NoError(t, err, "unexpected error")
}

func TestBuildRulesTree(t *testing.T) {
	BuildRulesTree(GetConf())
}
