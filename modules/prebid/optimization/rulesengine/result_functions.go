package rulesengine

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// hookstage.BidderRequestPayload - maybe different?
type ResultFunction interface {
	AddChangeSet(*hookstage.ChangeSet[hookstage.BidderRequestPayload]) error
}

func NewResultFunctionFactory(name string, params json.RawMessage) (ResultFunction, error) {
	switch name {
	case "excludeBidders":
		return NewExcludeBidders(params)
	case "includeBidders":
		return NewIncludeBidders(params)
	case "logATag":
		return NewLogATag(params)
	default:
		return nil, fmt.Errorf("result function %s was not created", name)
	}
}

// used in both ExckudeBiddeers and IncludeBidders
type ResultFuncParams struct {
	Bidders        []string
	SeatNonBid     int
	AnalyticsValue string
	IfSyncedId     bool
}

//---------------ExcludeBidders--------------

func NewExcludeBidders(params json.RawMessage) (ResultFunction, error) {
	var excludeBiddersParams []ResultFuncParams
	if err := jsonutil.Unmarshal(params, &excludeBiddersParams); err != nil {
		return nil, err
	}
	return &ExcludeBidders{Args: excludeBiddersParams}, nil
}

type ExcludeBidders struct {
	Args []ResultFuncParams
}

// do we need to return error?
func (eb *ExcludeBidders) AddChangeSet(changeSet *hookstage.ChangeSet[hookstage.BidderRequestPayload]) error {
	//  create a change set which captures the changes we want to apply
	// this function should NOT perform any modifications to the request
	for _, arg := range eb.Args {
		// changeSet.BidderRequest().Bidders().Delete(arg.Bidders) - example
		changeSet.BidderRequest().BAdv().Update(arg.Bidders) // write mutation functions
	}
	return nil
}

//---------------IncludeBidders--------------

func NewIncludeBidders(params json.RawMessage) (ResultFunction, error) {
	var includeBiddersParams []ResultFuncParams
	if err := jsonutil.Unmarshal(params, &includeBiddersParams); err != nil {
		return nil, err
	}
	return &IncludeBidders{Args: includeBiddersParams}, nil
}

type IncludeBidders struct {
	Args []ResultFuncParams
}

// do we need to return error?
func (eb *IncludeBidders) AddChangeSet(changeSet *hookstage.ChangeSet[hookstage.BidderRequestPayload]) error {
	//  create a change set which captures the changes we want to apply
	// this function should NOT perform any modifications to the request
	for _, arg := range eb.Args {
		// changeSet.BidderRequest().Bidders().Add(arg.Bidders) - example
		changeSet.BidderRequest().BAdv().Update(arg.Bidders) // write mutation functions
	}
	return nil
}

// ---------------LogATag--------------

type LogATagParams struct {
	AnalyticsValue string
}

func NewLogATag(params json.RawMessage) (ResultFunction, error) {
	var logATagParams LogATagParams
	if err := jsonutil.Unmarshal(params, &logATagParams); err != nil {
		return nil, err
	}
	return &LogATag{Args: logATagParams}, nil
}

type LogATag struct {
	Args LogATagParams
}

// do we need to return error?
func (lt *LogATag) AddChangeSet(changeSet *hookstage.ChangeSet[hookstage.BidderRequestPayload]) error {
	//  create a change set which captures the changes we want to apply
	// this function should NOT perform any modifications to the request

	// changeSet.BidderRequest().AnalyticsKey().Update(lt.AnalyticsValue) - example
	// changeSet.BidderRequest().ModelVersion().Update(lt.AnalyticsValue) - example
	changeSet.BidderRequest().BAdv().Update([]string{}) // write mutation functions

	return nil
}
