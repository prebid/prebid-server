package rulesengine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/v4/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/rules"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// ProcessedAuctionResultFunc is a type alias for a result function that runs in the processed auction request stage.
type ProcessedAuctionResultFunc = rules.ResultFunction[openrtb_ext.RequestWrapper, ProcessedAuctionHookResult]

const (
	ExcludeBiddersName = "excludeBidders"
	IncludeBiddersName = "includeBidders"
)

type FilterType string

const (
	FilterTypeGeo        FilterType = "geo"
	FilterTypeDatacenter FilterType = "datacenter"
	FilterTypeChannel    FilterType = "channel"
	FilterTypeIdentity   FilterType = "identity"
	FilterTypeFPD        FilterType = "fpd"
	FilterTypePrivacy    FilterType = "privacy"
	FilterTypeTraffic    FilterType = "traffic"
	FilterTypeUnknown    FilterType = "unknown"
)

// NewProcessedAuctionRequestResultFunction is a factory function that creates a new result function based on the provided name and parameters.
// It returns an error if the function name is not recognized or if there is an issue with the parameters.
// The function name is case insensitive.
// The parameters are expected to be in JSON format and will be unmarshalled into the appropriate struct.
// The function returns a rules.ResultFunction that can be used to modify the ProcessedAuctionRequestPayload in the ChangeSet.
// The function is used to create result functions for the rules engine in the Prebid Server.
func NewProcessedAuctionRequestResultFunction(name string, params json.RawMessage) (ProcessedAuctionResultFunc, error) {
	//TODO: make case insensitive converting to lower case
	switch name {
	case ExcludeBiddersName:
		return NewExcludeBidders(params)
	case IncludeBiddersName:
		return NewIncludeBidders(params)
	default:
		return nil, fmt.Errorf("result function %s was not created", name)
	}
}

// NewExcludeBidders is a factory function that creates a new ExcludeBidders result function.
// It takes a JSON raw message as input, unmarshals it into a slice of ResultFuncParams,
// and returns an ExcludeBidders instance.
// The function returns an error if there is an issue with the unmarshalling process.
// The ExcludeBidders function is used to modify the ProcessedAuctionRequestPayload in the ChangeSet.
func NewExcludeBidders(params json.RawMessage) (ProcessedAuctionResultFunc, error) {
	var excludeBiddersParams config.ResultFuncParams
	if err := jsonutil.Unmarshal(params, &excludeBiddersParams); err != nil {
		return nil, err
	}

	if len(excludeBiddersParams.Bidders) == 0 {
		return nil, errors.New("excludeBidders requires at least one bidder to be specified")
	}
	return &ExcludeBidders{Args: excludeBiddersParams}, nil
}

// ExcludeBidders is a struct that holds parameters for excluding bidders in the rules engine.
type ExcludeBidders struct {
	Args config.ResultFuncParams
}

// Call is a method that applies the changes specified in the ExcludeBidders instance to the provided ChangeSet by creating a mutation.
func (eb *ExcludeBidders) Call(req *openrtb_ext.RequestWrapper, result *ProcessedAuctionHookResult, meta rules.ResultFunctionMeta) error {
	excludedBidders := make(map[string]struct{})
	for _, bidderName := range eb.Args.Bidders {
		excludedBidders[bidderName] = struct{}{} // Ensure the bidder is included in the allowed bidders
	}

	result.HookResult.ChangeSet.ProcessedAuctionRequest().Bidders().Delete(excludedBidders)
	result.HookResult.SeatNonBid = appendUniqueFilteredSeatNonBids(result.HookResult.SeatNonBid, filteredSeatNonBids(req, eb.Args.Bidders, filterTypeFromMeta(meta)))
	return nil
}

func (eb *ExcludeBidders) Name() string {
	return ExcludeBiddersName
}

// NewIncludeBidders is a factory function that creates a new IncludeBidders result function.
// It takes a JSON raw message as input, unmarshals it into a slice of ResultFuncParams,
// and returns an IncludeBidders instance.
// The function returns an error if there is an issue with the unmarshalling process.
// The IncludeBidders function is used to modify the ProcessedAuctionRequestPayload in the ChangeSet.
func NewIncludeBidders(params json.RawMessage) (ProcessedAuctionResultFunc, error) {
	var includeBiddersParams config.ResultFuncParams
	if err := jsonutil.Unmarshal(params, &includeBiddersParams); err != nil {
		return nil, err
	}
	if len(includeBiddersParams.Bidders) == 0 {
		return nil, errors.New("includeBidders requires at least one bidder to be specified")
	}
	return &IncludeBidders{Args: includeBiddersParams}, nil
}

// IncludeBidders is a struct that holds parameters for including bidders in the rules engine.
type IncludeBidders struct {
	Args config.ResultFuncParams
}

// Call is a method that applies the changes specified in the IncludeBidders instance to the provided ChangeSet by creating a mutation.
func (ib *IncludeBidders) Call(req *openrtb_ext.RequestWrapper, result *ProcessedAuctionHookResult, meta rules.ResultFunctionMeta) error {
	for _, bidderName := range ib.Args.Bidders {
		result.AllowedBidders[bidderName] = struct{}{} // Ensure the bidder is included in the allowed bidders
	}
	result.IncludeContexts = append(result.IncludeContexts, meta)
	return nil
}

func (ib *IncludeBidders) Name() string {
	return IncludeBiddersName
}

func appendInclusionWarnings(req *openrtb_ext.RequestWrapper, result *ProcessedAuctionHookResult) {
	if len(result.AllowedBidders) == 0 || len(result.IncludeContexts) == 0 {
		return
	}

	allowed := make([]string, 0, len(result.AllowedBidders))
	for bidder := range result.AllowedBidders {
		allowed = append(allowed, bidder)
	}

	removedBidders := biddersNotAllowed(req, result.AllowedBidders)
	if len(removedBidders) == 0 {
		return
	}
	meta := result.IncludeContexts[len(result.IncludeContexts)-1]
	warning := fmt.Sprintf("Bidders [%s] were removed from the request by the rules engine include list [%s]", strings.Join(removedBidders, ", "), strings.Join(allowed, ", "))
	result.HookResult.Warnings = append(result.HookResult.Warnings, warning)
	result.HookResult.SeatNonBid = appendUniqueFilteredSeatNonBids(result.HookResult.SeatNonBid, filteredSeatNonBids(req, removedBidders, filterTypeFromMeta(meta)))
}

// Keep filtered statuses to one entry per bidder even if multiple rules remove the same bidder.
func appendUniqueFilteredSeatNonBids(existing []openrtb_ext.SeatNonBid, additions []openrtb_ext.SeatNonBid) []openrtb_ext.SeatNonBid {
	seen := make(map[string]struct{}, len(existing)+len(additions))
	for _, seatNonBid := range existing {
		for _, nonBid := range seatNonBid.NonBid {
			if nonBid.ImpId != "" || nonBid.StatusCode != 200 || nonBid.Ext == nil {
				continue
			}
			seen[filteredSeatNonBidKey(seatNonBid.Seat)] = struct{}{}
		}
	}
	for _, seatNonBid := range additions {
		if len(seatNonBid.NonBid) == 0 {
			continue
		}
		nonBid := seatNonBid.NonBid[0]
		if nonBid.Ext == nil {
			existing = append(existing, seatNonBid)
			continue
		}
		key := filteredSeatNonBidKey(seatNonBid.Seat)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		existing = append(existing, seatNonBid)
	}
	return existing
}

func filteredSeatNonBidKey(seat string) string {
	return seat
}

// Rules filtering is bidder/request scoped today, so emit one status per filtered bidder
// without impid instead of repeating the same reason for every impression.
func filteredSeatNonBids(req *openrtb_ext.RequestWrapper, bidders []string, filterType FilterType) []openrtb_ext.SeatNonBid {
	bidderSet := make(map[string]struct{}, len(bidders))
	for _, bidder := range bidders {
		bidderSet[bidder] = struct{}{}
	}

	filteredSeats := make(map[string]struct{}, len(bidders))
	if req == nil {
		return nil
	}
	for _, impWrapper := range req.GetImp() {
		impExt, err := impWrapper.GetImpExt()
		if err != nil {
			continue
		}
		impPrebid := impExt.GetPrebid()
		if impPrebid == nil {
			continue
		}
		for bidder := range impPrebid.Bidder {
			if _, ok := bidderSet[bidder]; !ok {
				continue
			}
			filteredSeats[bidder] = struct{}{}
		}
	}

	seatNonBids := make([]openrtb_ext.SeatNonBid, 0, len(filteredSeats))
	for seat := range filteredSeats {
		seatNonBids = append(seatNonBids, openrtb_ext.SeatNonBid{
			Seat: seat,
			NonBid: []openrtb_ext.NonBid{{
				StatusCode: 200,
				Ext: &openrtb_ext.NonBidExt{Prebid: openrtb_ext.ExtResponseNonBidPrebid{
					Type: string(filterType),
				}},
			}},
		})
	}
	return seatNonBids
}

func biddersNotAllowed(req *openrtb_ext.RequestWrapper, allowed map[string]struct{}) []string {
	if req == nil {
		return nil
	}
	removed := make(map[string]struct{})
	for _, impWrapper := range req.GetImp() {
		impExt, err := impWrapper.GetImpExt()
		if err != nil {
			continue
		}
		impPrebid := impExt.GetPrebid()
		if impPrebid == nil {
			continue
		}
		for bidder := range impPrebid.Bidder {
			if _, ok := allowed[bidder]; !ok {
				removed[bidder] = struct{}{}
			}
		}
	}

	bidders := make([]string, 0, len(removed))
	for bidder := range removed {
		bidders = append(bidders, bidder)
	}
	return bidders
}

func filterTypeFromMeta(meta rules.ResultFunctionMeta) FilterType {
	for _, step := range meta.SchemaFunctionResults {
		switch step.FuncName {
		case rules.DeviceCountry, rules.DeviceCountryIn:
			return FilterTypeGeo
		case rules.DataCenter, rules.DataCenterIn:
			return FilterTypeDatacenter
		case rules.Channel:
			return FilterTypeChannel
		case rules.EidAvailable, rules.EidIn:
			return FilterTypeIdentity
		case rules.FpdAvailable, rules.UserFpdAvailable:
			return FilterTypeFPD
		case rules.TcfInScope, rules.GppSidAvailable, rules.GppSidIn:
			return FilterTypePrivacy
		case rules.Percent:
			return FilterTypeTraffic
		}
	}
	return FilterTypeUnknown
}
