package rulesengine

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"slices"

	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/rules"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// ProcessedAuctionResultFunc is a type alias for a result function that runs in the processed auction request stage.
type ProcessedAuctionResultFunc = rules.ResultFunction[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]

const (
	ExcludeBiddersName = "excludeBidders"
	IncludeBiddersName = "includeBidders"
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
func (eb *ExcludeBidders) Call(req *openrtb_ext.RequestWrapper, result *hs.HookResult[hs.ProcessedAuctionRequestPayload], meta rules.ResultFunctionMeta) error {
	//  create a change set which captures the changes we want to apply
	// this function should NOT perform any modifications to the request

	// build map[impId] to map [bidder] to bidder params
	impIdToBidders, err := buildExcludeBidders(req, eb.Args.Bidders)
	if err != nil {
		return err
	}

	result.ChangeSet.ProcessedAuctionRequest().Bidders().Update(impIdToBidders)

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
func (ib *IncludeBidders) Call(req *openrtb_ext.RequestWrapper, result *hs.HookResult[hs.ProcessedAuctionRequestPayload], meta rules.ResultFunctionMeta) error {
	//  create a change set which captures the changes we want to apply
	// this function should NOT perform any modifications to the request

	// build map[impId] to map [bidder] to bidder params
	impIdToBidders, err := buildIncludeBidders(req, ib.Args.Bidders)
	if err != nil {
		return err
	}

	result.ChangeSet.ProcessedAuctionRequest().Bidders().Update(impIdToBidders)

	return nil
}

func (ib *IncludeBidders) Name() string {
	return IncludeBiddersName
}

func buildIncludeBidders(req *openrtb_ext.RequestWrapper, argBidders []string) (map[string]map[string]json.RawMessage, error) {
	impIdToBidders := make(map[string]map[string]json.RawMessage)
	for _, impWrapper := range req.GetImp() {
		impExt, impExtErr := impWrapper.GetImpExt()
		if impExtErr != nil {
			return impIdToBidders, impExtErr
		}
		impPrebid := impExt.GetPrebid()
		if impPrebid == nil {
			return nil, fmt.Errorf("impExt for imp %s does not contain prebid extension", impWrapper.ID)
		}
		impBidders := impPrebid.Bidder

		resultImpBidders := make(map[string]json.RawMessage)
		for _, bidder := range argBidders {
			// add only bidders from argBidders
			if _, ok := impBidders[bidder]; ok {
				resultImpBidders[bidder] = impBidders[bidder]
			}
		}
		impIdToBidders[impWrapper.ID] = resultImpBidders
	}
	return impIdToBidders, nil
}

func buildExcludeBidders(req *openrtb_ext.RequestWrapper, argBidders []string) (map[string]map[string]json.RawMessage, error) {
	impIdToBidders := make(map[string]map[string]json.RawMessage)
	for _, impWrapper := range req.GetImp() {
		impExt, impExtErr := impWrapper.GetImpExt()
		if impExtErr != nil {
			return impIdToBidders, impExtErr
		}
		impPrebid := impExt.GetPrebid()
		if impPrebid == nil {
			return nil, fmt.Errorf("impExt for imp %s does not contain prebid extension", impWrapper.ID)
		}
		impBidders := impPrebid.Bidder

		resultImpBidders := make(map[string]json.RawMessage)

		for bidderName, bidderData := range impBidders {
			// do not add bidders from argBidders
			if contains := slices.Contains(argBidders, bidderName); !contains {
				resultImpBidders[bidderName] = bidderData
			}
		}

		impIdToBidders[impWrapper.ID] = resultImpBidders
	}
	return impIdToBidders, nil
}
