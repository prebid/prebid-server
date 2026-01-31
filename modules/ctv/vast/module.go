package vast

import (
	"github.com/prebid/prebid-server/v3/modules/ctv/vast/model"
)

// Module is the entry point for the CTV VAST module
// This orchestrates the entire pipeline: select -> enrich -> format
type Module struct {
	selector  BidSelector
	enricher  Enricher
	formatter Formatter
}

// NewModule creates a new VAST module with the given components
func NewModule(selector BidSelector, enricher Enricher, formatter Formatter) *Module {
	return &Module{
		selector:  selector,
		enricher:  enricher,
		formatter: formatter,
	}
}

// Process is the main entry point that runs the complete pipeline
// It takes an OpenRTB request/response pair and produces VAST XML
func (m *Module) Process(req interface{}, resp interface{}, cfg ReceiverConfig) (*VastResult, error) {
	// TODO: Implement pipeline orchestration
	// 1. Call selector.Select() to choose bids
	// 2. For each selected bid:
	//    a. Parse existing VAST from bid.adm (if present)
	//    b. Create new VastAd if no VAST exists
	//    c. Call enricher.Enrich() to add metadata
	// 3. Call formatter.Format() to produce final XML
	// 4. Collect warnings and errors throughout
	// 5. Return VastResult with XML, selected bids, warnings, errors
	
	return &VastResult{
		VastXML:  model.BuildNoAdVast(cfg.VastVersionDefault),
		NoAd:     true,
		Warnings: nil,
		Errors:   nil,
		Selected: nil,
	}, nil
}
