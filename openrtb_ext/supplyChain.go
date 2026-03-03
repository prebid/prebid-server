package openrtb_ext

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

func cloneSupplyChain(schain *openrtb2.SupplyChain) *openrtb2.SupplyChain {
	if schain == nil {
		return nil
	}
	clone := *schain
	clone.Nodes = make([]openrtb2.SupplyChainNode, len(schain.Nodes))
	for i, node := range schain.Nodes {
		clone.Nodes[i] = node
		clone.Nodes[i].HP = ptrutil.Clone(schain.Nodes[i].HP)
	}

	return &clone

}
