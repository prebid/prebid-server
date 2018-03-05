package openx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const uri = "http://rtb.openx.net/prebid"

type OpenxAdapter struct {
}

type openxImpExt struct {
	CustomParams map[string]interface{} `json:"customParams,omitempty"`
}

type openxReqExt struct {
	DelDomain string `json:"delDomain"`
}

func (a *OpenxAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	var errs []error
	var validImps []openrtb.Imp
	var reqExt openxReqExt

	for _, imp := range request.Imp {
		if err := preprocess(&imp, &reqExt); err != nil {
			errs = append(errs, err)
			continue
		}
		validImps = append(validImps, imp)
	}

	request.Imp = validImps
	// If all the imps were malformed, don't bother making a server call with no impressions.
	if len(request.Imp) == 0 {
		return nil, errs
	}

	var err error
	request.Ext, err = json.Marshal(reqExt)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     uri,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

// Mutate the imp to get it ready to send to openx.
func preprocess(imp *openrtb.Imp, reqExt *openxReqExt) error {
	// We only support banner impressions for now.
	if imp.Video != nil || imp.Native != nil || imp.Audio != nil {
		return fmt.Errorf("OpenX doesn't support video, audio or native Imps. Ignoring Imp ID=%s", imp.ID)
	}

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return err
	}

	var openxExt openrtb_ext.ExtImpOpenx
	if err := json.Unmarshal(bidderExt.Bidder, &openxExt); err != nil {
		return err
	}

	reqExt.DelDomain = openxExt.DelDomain

	imp.TagID = openxExt.Unit
	imp.BidFloor = openxExt.CustomFloor
	imp.Ext = nil

	if openxExt.CustomParams != nil {
		impExt := openxImpExt{
			CustomParams: openxExt.CustomParams,
		}
		var err error
		if imp.Ext, err = json.Marshal(impExt); err != nil {
			return err
		}
	}

	return nil
}

func (a *OpenxAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) ([]*adapters.TypedBid, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bids := make([]*adapters.TypedBid, 0, 5)

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bids = append(bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}
	return bids, nil
}

func NewOpenxBidder() *OpenxAdapter {
	return &OpenxAdapter{}
}
