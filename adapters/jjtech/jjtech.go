package jjtech

import (
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint string
}

// jjtechImpExt is the imp.ext shape JJTech's bidding server expects.
type jjtechImpExt struct {
	JJTech openrtb_ext.ExtImpJJTech `json:"jjtech"`
}

// Builder builds a new instance of the JJTech adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: cfg.Endpoint}, nil
}

// MakeRequests batches every valid impression into a single call to JJTech's bidding server,
// replacing the Prebid Server imp.ext.bidder params with JJTech's own imp.ext.jjtech shape.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	imps := make([]openrtb2.Imp, 0, len(request.Imp))

	for _, imp := range request.Imp {
		params, err := parseImpParams(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		ext, err := jsonutil.Marshal(jjtechImpExt{JJTech: params})
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// imp is a copy, so this leaves the caller's request untouched.
		imp.Ext = ext
		imps = append(imps, imp)
	}

	if len(imps) == 0 {
		return nil, errs
	}

	outgoing := *request
	outgoing.Imp = imps

	body, err := jsonutil.Marshal(outgoing)
	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(imps),
	}}, errs
}

func parseImpParams(imp openrtb2.Imp) (openrtb_ext.ExtImpJJTech, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb_ext.ExtImpJJTech{}, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: failed to parse imp.ext: %s", imp.ID, err.Error()),
		}
	}

	var params openrtb_ext.ExtImpJJTech
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &params); err != nil {
		return openrtb_ext.ExtImpJJTech{}, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: failed to parse jjtech params: %s", imp.ID, err.Error()),
		}
	}

	if params.PlacementID == "" {
		return openrtb_ext.ExtImpJJTech{}, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: placementId is required", imp.ID),
		}
	}

	return params, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return nil, nil
}
