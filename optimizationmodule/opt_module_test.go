package optimizationmodule

import (
	"encoding/json"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExecuteRulesRecursive(t *testing.T) {

	rules := BuildTestRules()
	rw := BuildTestRequestWrapper()
	err := rules.Execute(rw)
	assert.Equal(t, "127.0.0.1", rw.Device.IP)
	assert.NoError(t, err, "unexpected error")
}

func TestExecuteRulesRecursiveFullConfig(t *testing.T) {

	rules := BuildRulesTree(GetConf())
	rw := BuildTestRequestWrapper()
	err := rules.Execute(rw)
	assert.Equal(t, "127.0.0.1", rw.Device.IP)
	assert.NoError(t, err, "unexpected error")
}

func TestExecuteRulesFlat(t *testing.T) {

	rules := BuildTestRules()
	rw := BuildTestRequestWrapper()
	_, err := ExecuteFlat(&rules, rw)
	assert.Equal(t, "127.0.0.1", rw.Device.IP)
	assert.NoError(t, err, "unexpected error")
}

func TestExecuteRulesFlatFullConfig(t *testing.T) {

	rules := BuildRulesTree(GetConf())
	rw := BuildTestRequestWrapper()
	_, err := ExecuteFlat(rules, rw)
	assert.Equal(t, "127.0.0.1", rw.Device.IP)
	assert.NoError(t, err, "unexpected error")
}

func TestBuildRulesTree(t *testing.T) {
	BuildRulesTree(GetConf())
}

func BuildTestRules() Rules {
	rules := Rules{
		Root: &Node{
			Function: NewDeviceCountry([]string{"USA"}),
			Children: map[string]*Node{
				"true":  &Node{Function: NewSetDevIp(json.RawMessage(`127.0.0.1`))}, //can have children
				"false": &Node{Function: NewSetDevIp(json.RawMessage(`127.0.0.2`))}, // can have children
			},
		},
	}
	return rules
}

func BuildTestRequestWrapper() *openrtb_ext.RequestWrapper {
	rw := &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			Device: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Country: "USA",
					Region:  "us-east",
					City:    "amp",
				},
				IP: "0.0.0.0",
			},
		}}
	return rw
}
