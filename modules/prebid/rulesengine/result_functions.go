package rulesengine

import (
	"encoding/json"
	"fmt"

	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/rules"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	ExcludeBiddersName = "excludeBidders"
	IncludeBiddersName = "includeBidders"
	LogATagName        = "logATag"
)

// NewProcessedAuctionRequestResultFunction is a factory function that creates a new result function based on the provided name and parameters.
// It returns an error if the function name is not recognized or if there is an issue with the parameters.
// The function name is case insensitive.
// The parameters are expected to be in JSON format and will be unmarshalled into the appropriate struct.
// The function returns a rules.ResultFunction that can be used to modify the ProcessedAuctionRequestPayload in the ChangeSet.
// The function is used to create result functions for the rules engine in the Prebid Server.
func NewProcessedAuctionRequestResultFunction(name string, params json.RawMessage) (rules.ResultFunction[hs.ChangeSet[hs.ProcessedAuctionRequestPayload]], error) {
	//TODO: make case insensitive converting to lower case
	switch name {
	case ExcludeBiddersName:
		return NewExcludeBidders(params)
	case IncludeBiddersName:
		return NewIncludeBidders(params)
	case LogATagName:
		return NewLogATag(params)
	default:
		return nil, fmt.Errorf("result function %s was not created", name)
	}
}

// ResultFuncParams is a struct that holds parameters for result functions and is used in ExcludeBidders and IncludeBidders.
type ResultFuncParams struct {
	Bidders        []string
	SeatNonBid     int
	AnalyticsValue string
	IfSyncedId     bool
}

// NewExcludeBidders is a factory function that creates a new ExcludeBidders result function.
// It takes a JSON raw message as input, unmarshals it into a slice of ResultFuncParams,
// and returns an ExcludeBidders instance.
// The function returns an error if there is an issue with the unmarshalling process.
// The ExcludeBidders function is used to modify the ProcessedAuctionRequestPayload in the ChangeSet.
func NewExcludeBidders(params json.RawMessage) (rules.ResultFunction[hs.ChangeSet[hs.ProcessedAuctionRequestPayload]], error) {
	var excludeBiddersParams []ResultFuncParams
	if err := jsonutil.Unmarshal(params, &excludeBiddersParams); err != nil {
		return nil, err
	}
	return &ExcludeBidders{Args: excludeBiddersParams, funcName: ExcludeBiddersName}, nil
}

// ExcludeBidders is a struct that holds parameters for excluding bidders in the rules engine.
type ExcludeBidders struct {
	funcName string
	Args     []ResultFuncParams
}

// Call is a method that applies the changes specified in the ExcludeBidders instance to the provided ChangeSet by creating a mutation.
func (eb *ExcludeBidders) Call(changeSet *hs.ChangeSet[hs.ProcessedAuctionRequestPayload], schemaFunctionsResults map[string]string) error {
	//  create a change set which captures the changes we want to apply
	// this function should NOT perform any modifications to the request
	for _, arg := range eb.Args {
		// changeSet.BidderRequest().Bidders().Delete(arg.Bidders) - example
		changeSet.BidderRequest().BAdv().Update(arg.Bidders) // write mutation functions
	}
	return nil
}

// NewIncludeBidders is a factory function that creates a new IncludeBidders result function.
// It takes a JSON raw message as input, unmarshals it into a slice of ResultFuncParams,
// and returns an IncludeBidders instance.
// The function returns an error if there is an issue with the unmarshalling process.
// The IncludeBidders function is used to modify the ProcessedAuctionRequestPayload in the ChangeSet.
func NewIncludeBidders(params json.RawMessage) (rules.ResultFunction[hs.ChangeSet[hs.ProcessedAuctionRequestPayload]], error) {
	var includeBiddersParams []ResultFuncParams
	if err := jsonutil.Unmarshal(params, &includeBiddersParams); err != nil {
		return nil, err
	}
	return &IncludeBidders{Args: includeBiddersParams, funcName: IncludeBiddersName}, nil
}

// IncludeBidders is a struct that holds parameters for including bidders in the rules engine.
type IncludeBidders struct {
	funcName string
	Args     []ResultFuncParams
}

// Call is a method that applies the changes specified in the IncludeBidders instance to the provided ChangeSet by creating a mutation.
func (eb *IncludeBidders) Call(changeSet *hs.ChangeSet[hs.ProcessedAuctionRequestPayload], schemaFunctionsResults map[string]string) error {
	//  create a change set which captures the changes we want to apply
	// this function should NOT perform any modifications to the request

	if adUnitCode, ok := schemaFunctionsResults[rules.AdUnitCode]; ok {
		if adUnitCode != "*" { // wildcard
			// add comparison logic
		}
	}

	if mediaType, ok := schemaFunctionsResults[rules.MediaTypes]; ok {
		if mediaType != "*" { // wildcard
			// add comparison logic
		}
	}

	/*if len{analyticsKey} > 0{
		//create an analytics tag
	}*/

	for _, arg := range eb.Args {
		if arg.IfSyncedId {

		}

		// changeSet.BidderRequest().Bidders().Add(arg.Bidders) - example
		changeSet.BidderRequest().BAdv().Update(arg.Bidders) // write mutation functions
	}
	return nil
}

// LogATagParams is a struct that holds parameters for the LogATag result function.
type LogATagParams struct {
	AnalyticsValue string
}

// NewLogATag is a factory function that creates a new LogATag result function.
// It takes a JSON raw message as input, unmarshals it into a LogATagParams struct,
// and returns a LogATag instance.
// The function returns an error if there is an issue with the unmarshalling process.
// The LogATag function is used to modify the ProcessedAuctionRequestPayload in the ChangeSet.
func NewLogATag(params json.RawMessage) (rules.ResultFunction[hs.ChangeSet[hs.ProcessedAuctionRequestPayload]], error) {
	var logATagParams LogATagParams
	if err := jsonutil.Unmarshal(params, &logATagParams); err != nil {
		return nil, err
	}
	return &LogATag{Args: logATagParams, funcName: LogATagName}, nil
}

// LogATag is a struct that holds parameters for the LogATag result function.
type LogATag struct {
	funcName string
	Args     LogATagParams
}

// Call is a method that applies the changes specified in the LogATag instance to the provided ChangeSet by creating a mutation
func (lt *LogATag) Call(changeSet *hs.ChangeSet[hs.ProcessedAuctionRequestPayload], schemaFunctionsResults map[string]string) error {
	//  create a change set which captures the changes we want to apply
	// this function should NOT perform any modifications to the request

	// changeSet.BidderRequest().AnalyticsKey().Update(lt.AnalyticsValue) - example
	// changeSet.BidderRequest().ModelVersion().Update(lt.AnalyticsValue) - example
	changeSet.BidderRequest().BAdv().Update([]string{}) // write mutation functions

	return nil
}
