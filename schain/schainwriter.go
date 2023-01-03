package schain

import (
	"encoding/json"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
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
func (w SChainWriter) Write(req *openrtb2.BidRequest, bidder string) {
	const sChainWildCard = "*"
	var selectedSChain *openrtb2.SupplyChain

	wildCardSChain := w.sChainsByBidder[sChainWildCard]
	bidderSChain := w.sChainsByBidder[bidder]

	// source should not be modified
	if bidderSChain == nil && wildCardSChain == nil && w.hostSChainNode == nil {
		return
	}

	selectedSChain = &openrtb2.SupplyChain{Ver: "1.0"}

	if bidderSChain != nil {
		selectedSChain = bidderSChain
	} else if wildCardSChain != nil {
		selectedSChain = wildCardSChain
	}

	schain := openrtb_ext.ExtRequestPrebidSChain{
		SChain: *selectedSChain,
	}

	if req.Source == nil {
		req.Source = &openrtb2.Source{}
	} else {
		sourceCopy := *req.Source
		req.Source = &sourceCopy
	}

	if w.hostSChainNode != nil {
		schain.SChain.Nodes = append(schain.SChain.Nodes, *w.hostSChainNode)
	}

	sourceExt, err := json.Marshal(schain)
	if err == nil {
		req.Source.Ext = sourceExt
	}

}

// extPrebidSChainExists checks if an schain exists in the ORTB 2.5 req.ext.prebid.schain location
func extPrebidSChainExists(reqExt *openrtb_ext.ExtRequest) bool {
	return reqExt != nil && reqExt.Prebid.SChains != nil
}
