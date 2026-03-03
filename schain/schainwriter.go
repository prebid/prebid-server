package schain

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// NewSChainWriter creates an ORTB 2.5 schain writer instance
func NewSChainWriter(reqExt *openrtb_ext.ExtRequest, hostSChainNode *openrtb2.SupplyChainNode) (*SChainWriter, error) {
	if !extPrebidSChainExists(reqExt) {
		return &SChainWriter{hostSChainNode: hostSChainNode}, nil
	}

	sChainsByBidder, err := BidderToPrebidSChains(reqExt.Prebid.SChains)
	if err != nil {
		return nil, err
	}

	writer := SChainWriter{
		sChainsByBidder: sChainsByBidder,
		hostSChainNode:  hostSChainNode,
	}
	return &writer, nil
}

// SChainWriter is used to write the appropriate schain for a particular bidder defined in the ORTB 2.5 multi-schain
// location (req.ext.prebid.schain) to the ORTB 2.5 location (req.source.ext)
type SChainWriter struct {
	sChainsByBidder map[string]*openrtb2.SupplyChain
	hostSChainNode  *openrtb2.SupplyChainNode
}

// Write selects an schain from the multi-schain ORTB 2.5 location (req.ext.prebid.schains) for the specified bidder
// and copies it to the ORTB 2.5 location (req.source.ext). If no schain exists for the bidder in the multi-schain
// location and no wildcard schain exists, the request is not modified.
func (w SChainWriter) Write(reqWrapper *openrtb_ext.RequestWrapper, bidder string) {
	const sChainWildCard = "*"
	var selectedSChain openrtb2.SupplyChain

	wildCardSChain := w.sChainsByBidder[sChainWildCard]
	bidderSChain := w.sChainsByBidder[bidder]

	// source should not be modified
	if bidderSChain == nil && wildCardSChain == nil && w.hostSChainNode == nil {
		return
	}

	selectedSChain = openrtb2.SupplyChain{Ver: "1.0"}

	if bidderSChain != nil {
		selectedSChain = *bidderSChain
	} else if wildCardSChain != nil {
		selectedSChain = *wildCardSChain
	}

	if reqWrapper.Source == nil {
		reqWrapper.Source = &openrtb2.Source{}
	} else {
		// Copy Source to avoid shared memory issues.
		// Source may be modified differently for different bidders in request
		sourceCopy := *reqWrapper.Source
		reqWrapper.Source = &sourceCopy
	}

	reqWrapper.Source.SChain = &selectedSChain

	if w.hostSChainNode != nil {
		reqWrapper.Source.SChain.Nodes = append(reqWrapper.Source.SChain.Nodes, *w.hostSChainNode)
	}
}

// extPrebidSChainExists checks if an schain exists in the ORTB 2.5 req.ext.prebid.schain location
func extPrebidSChainExists(reqExt *openrtb_ext.ExtRequest) bool {
	return reqExt != nil && reqExt.Prebid.SChains != nil
}
