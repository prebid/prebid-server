package rulesengine

import (
	"encoding/json"
	"fmt"

	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/rules"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// ProcessedAuctionResultFunc is a type alias for a result function that runs in the processed auction request stage.
type ProcessedAuctionResultFunc = rules.ResultFunction[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]

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
func NewProcessedAuctionRequestResultFunction(name string, params json.RawMessage) (ProcessedAuctionResultFunc, error) {
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
func NewExcludeBidders(params json.RawMessage) (ProcessedAuctionResultFunc, error) {
	var excludeBiddersParams []ResultFuncParams
	if err := jsonutil.Unmarshal(params, &excludeBiddersParams); err != nil {
		return nil, err
	}
	return &ExcludeBidders{Args: excludeBiddersParams}, nil
}

// ExcludeBidders is a struct that holds parameters for excluding bidders in the rules engine.
type ExcludeBidders struct {
	Args []ResultFuncParams
}

// Call is a method that applies the changes specified in the ExcludeBidders instance to the provided ChangeSet by creating a mutation.
func (eb *ExcludeBidders) Call(req *openrtb_ext.RequestWrapper, changeSet *hs.ChangeSet[hs.ProcessedAuctionRequestPayload], funcMeta rules.ResultFuncMetadata) error {
	//  create a change set which captures the changes we want to apply
	// this function should NOT perform any modifications to the request
	for _, arg := range eb.Args {
		// changeSet.BidderRequest().Bidders().Delete(arg.Bidders) - example
		changeSet.BidderRequest().BAdv().Update(arg.Bidders) // write mutation functions
	}
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
	var includeBiddersParams []ResultFuncParams
	if err := jsonutil.Unmarshal(params, &includeBiddersParams); err != nil {
		return nil, err
	}
	return &IncludeBidders{Args: includeBiddersParams}, nil
}

// IncludeBidders is a struct that holds parameters for including bidders in the rules engine.
type IncludeBidders struct {
	Args []ResultFuncParams
}

// Call is a method that applies the changes specified in the IncludeBidders instance to the provided ChangeSet by creating a mutation.
func (eb *IncludeBidders) Call(req *openrtb_ext.RequestWrapper, changeSet *hs.ChangeSet[hs.ProcessedAuctionRequestPayload], funcMeta rules.ResultFuncMetadata) error {
	//  create a change set which captures the changes we want to apply
	// this function should NOT perform any modifications to the request

	for _, meta := range funcMeta.SchemaFunctionResults {
		if meta.FuncName == rules.AdUnitCode {
			if meta.FuncResult != "*" { // wildcard
				// add comparison logic
			}
		}
		if meta.FuncName == rules.MediaTypes {
			if meta.FuncResult != "*" { // wildcard
				// add comparison logic
			}
		}
	}

	/*if len{analyticsKey} > 0{
		//create an analytics tag
	}*/

	// maybe merge all args into one to remove for loop - in res funct constructor NewIncludeBidders
	for _, arg := range eb.Args {
		if arg.IfSyncedId {
			// possibly modify args.bidders
		}
		// build map[impId] to map [bidder] to bidder params
		impIdToBidders := make(map[string]map[string]json.RawMessage)

		changeSet.BidderRequest().Bidders().Update(impIdToBidders)
	}
	return nil
}

func (eb *IncludeBidders) Name() string {
	return IncludeBiddersName
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
func NewLogATag(params json.RawMessage) (ProcessedAuctionResultFunc, error) {
	var logATagParams LogATagParams
	if err := jsonutil.Unmarshal(params, &logATagParams); err != nil {
		return nil, err
	}
	return &LogATag{Args: logATagParams}, nil
}

// LogATag is a struct that holds parameters for the LogATag result function.
type LogATag struct {
	Args LogATagParams
}

// Call is a method that applies the changes specified in the LogATag instance to the provided ChangeSet by creating a mutation
func (lt *LogATag) Call(req *openrtb_ext.RequestWrapper, changeSet *hs.ChangeSet[hs.ProcessedAuctionRequestPayload], funcMeta rules.ResultFuncMetadata) error {
	//stub
	return nil
}

func (lt *LogATag) Name() string {
	return LogATagName
}
