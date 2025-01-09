package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestCloneSupplyChain(t *testing.T) {
	testCases := []struct {
		name       string
		schain     *openrtb2.SupplyChain
		schainCopy *openrtb2.SupplyChain                            // manual copy of above prebid object to verify against
		mutator    func(t *testing.T, schain *openrtb2.SupplyChain) // function to modify the prebid object
	}{
		{
			name:       "Nil", // Verify the nil case
			schain:     nil,
			schainCopy: nil,
			mutator:    func(t *testing.T, schain *openrtb2.SupplyChain) {},
		},
		{
			name: "General",
			schain: &openrtb2.SupplyChain{
				Complete: 2,
				Nodes: []openrtb2.SupplyChainNode{
					{
						SID:  "alpha",
						Name: "Johnny",
						HP:   ptrutil.ToPtr[int8](5),
						Ext:  json.RawMessage(`{}`),
					},
					{
						ASI:  "Oh my",
						Name: "Johnny",
						HP:   ptrutil.ToPtr[int8](5),
						Ext:  json.RawMessage(`{"samson"}`),
					},
				},
				Ver: "v2.5",
				Ext: json.RawMessage(`{"foo": "bar"}`),
			},
			schainCopy: &openrtb2.SupplyChain{
				Complete: 2,
				Nodes: []openrtb2.SupplyChainNode{
					{
						SID:  "alpha",
						Name: "Johnny",
						HP:   ptrutil.ToPtr[int8](5),
						Ext:  json.RawMessage(`{}`),
					},
					{
						ASI:  "Oh my",
						Name: "Johnny",
						HP:   ptrutil.ToPtr[int8](5),
						Ext:  json.RawMessage(`{"samson"}`),
					},
				},
				Ver: "v2.5",
				Ext: json.RawMessage(`{"foo": "bar"}`),
			},
			mutator: func(t *testing.T, schain *openrtb2.SupplyChain) {
				schain.Nodes[0].SID = "beta"
				schain.Nodes[1].HP = nil
				schain.Nodes[0].Ext = nil
				schain.Nodes = append(schain.Nodes, openrtb2.SupplyChainNode{SID: "Gamma"})
				schain.Complete = 0
				schain.Ext = json.RawMessage(`{}`)
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			clone := cloneSupplyChain(test.schain)
			test.mutator(t, test.schain)
			assert.Equal(t, test.schainCopy, clone)
		})
	}
}
