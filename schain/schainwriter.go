package schain

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// SChainWriter identifies the relevant schain for a given bidder in the original request and writes it
// to the ORTB 2.5 schain (req.source.ext) location in the bid request
type SChainWriter interface {
	Write(req *openrtb2.BidRequest, bidder string)
}

// NewSChainWriter gets the appropriate SChainWriter instance based on where the schains are defined
func NewSChainWriter(reqExt *openrtb_ext.ExtRequest, sourceExt *openrtb_ext.ExtSource) (SChainWriter, error) {
	if extPrebidSChainExists(reqExt) {
		return newORTBTwoFiveSChainWriter(reqExt.Prebid.SChains)
	}
	if sourceExtSChainExists(sourceExt) {
		return newORTBTwoFiveSChainWriter(nil)
	}
	if extSChainExists(reqExt) {
		return newORTBTwoFourSChainWriter(reqExt), nil
	}
	// no schains provided for this bidder so fall back to ORTB 2.5
	return newORTBTwoFiveSChainWriter(nil)
}

// newORTBTwoFiveSChainWriter creates an ORTB 2.5 schain writer instance
func newORTBTwoFiveSChainWriter(schains []*openrtb_ext.ExtRequestPrebidSChain) (SChainWriter, error) {
	writer := ORTBTwoFiveSChainWriter{}
	sChainsByBidder, err := BidderToPrebidSChains(schains)
	if err != nil {
		return nil, err
	}
	writer.sChainsByBidder = sChainsByBidder
	return writer, nil
}

// ORTBTwoFiveSChainWriter implements the SChainWriter interface and is used to write the appropriate schain
// for a particular bidder defined in the ORTB 2.5 multi-schain location (req.ext.prebid.schain) to the 
// ORTB 2.5 location (req.source.ext)
type ORTBTwoFiveSChainWriter struct {
	sChainsByBidder map[string]*openrtb_ext.ExtRequestPrebidSChainSChain
}

// Write selects an schain from the multi-schain ORTB 2.5 location (req.ext.prebid.schains) for the specified bidder
// and copies it to the ORTB 2.5 location (req.source.ext). If no schain exists for the bidder in the multi-schain 
// location and no wildcard schain exists, the request is not modified.
func (w ORTBTwoFiveSChainWriter) Write(req *openrtb2.BidRequest, bidder string) {
	const sChainWildCard = "*"
	var selectedSChain *openrtb_ext.ExtRequestPrebidSChainSChain

	wildCardSChain := w.sChainsByBidder[sChainWildCard]
	bidderSChain := w.sChainsByBidder[bidder]

	// source should not be modified
	if bidderSChain == nil && wildCardSChain == nil {
		return
	}

	if bidderSChain != nil {
		selectedSChain = bidderSChain
	} else {
		selectedSChain = wildCardSChain
	}

	if req.Source == nil {
		req.Source = &openrtb2.Source{}
	} else {
		sourceCopy := *req.Source
		req.Source = &sourceCopy
	}
	schain := openrtb_ext.ExtRequestPrebidSChain{
		SChain: *selectedSChain,
	}
	sourceExt, err := json.Marshal(schain)
	if err == nil {
		req.Source.Ext = sourceExt
	}
}

// newORTBTwoFourSChainWriter creates an ORTB 2.4 schain writer instance.
func newORTBTwoFourSChainWriter(reqExt *openrtb_ext.ExtRequest) SChainWriter {
	return ORTBTwoFourSChainWriter{
		schain: reqExt.SChain,
	}
}

// ORTBTwoFourSChainWriter implements the SChainWriter interface and is used to write an schain
// defined in the ORTB 2.4 location (req.ext.schain) to the ORTB 2.5 location (req.source.ext)
type ORTBTwoFourSChainWriter struct {
	schain *openrtb_ext.ExtRequestPrebidSChainSChain
}

// Write copies an schain defined in the ORTB 2.4 location (req.ext.schain) to the ORTB 2.5 location (req.source.ext)
func (w ORTBTwoFourSChainWriter) Write(req *openrtb2.BidRequest, bidder string) {
	if w.schain == nil {
		return
	}

	if req.Source == nil {
		req.Source = &openrtb2.Source{}
	} else {
		sourceCopy := *req.Source
		req.Source = &sourceCopy
	}

	schain := openrtb_ext.ExtRequestPrebidSChain{
		SChain: *w.schain,
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

// sourceExtSChainExists checks if an schain exists in the ORTB 2.5 req.source.ext.schain location
func sourceExtSChainExists(reqSourceExt *openrtb_ext.ExtSource) bool {
	return reqSourceExt != nil && reqSourceExt.SChain != nil
}

// extSChainExists checks if an schain exists in the ORTB 2.4 req.ext.schain location
func extSChainExists(reqExt *openrtb_ext.ExtRequest) bool {
	return reqExt != nil && reqExt.SChain != nil
}
