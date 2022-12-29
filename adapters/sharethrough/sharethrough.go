package sharethrough

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/version"
)

var adapterVersion = "10.0"

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Sharethrough adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errors []error

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	modifiableSource := openrtb2.Source{}
	if request.Source != nil {
		modifiableSource = *request.Source
	}
	var sourceExt map[string]interface{}
	if err := json.Unmarshal(modifiableSource.Ext, &sourceExt); err == nil {
		sourceExt["str"] = adapterVersion
		sourceExt["version"] = version.Ver
	} else {
		sourceExt = map[string]interface{}{"str": adapterVersion, "version": version.Ver}
	}

	var err error
	if modifiableSource.Ext, err = json.Marshal(sourceExt); err != nil {
		errors = append(errors, err)
	}
	request.Source = &modifiableSource

	requestCopy := *request
	for _, imp := range request.Imp {
		// Extract Sharethrough Params
		var strImpExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &strImpExt); err != nil {
			errors = append(errors, err)
			continue
		}
		var strImpParams openrtb_ext.ExtImpSharethrough
		if err := json.Unmarshal(strImpExt.Bidder, &strImpParams); err != nil {
			errors = append(errors, err)
			continue
		}

		// Convert Floor into USD
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && !strings.EqualFold(imp.BidFloorCur, "USD") {
			convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
			if err != nil {
				return nil, []error{err}
			}
			imp.BidFloorCur = "USD"
			imp.BidFloor = convertedValue
		}

		// Relocate Custom Params
		imp.TagID = strImpParams.Pkey
		requestCopy.BCat = append(requestCopy.BCat, strImpParams.BCat...)
		requestCopy.BAdv = append(requestCopy.BAdv, strImpParams.BAdv...)

		requestCopy.Imp = []openrtb2.Imp{imp}

		requestJSON, err := json.Marshal(requestCopy)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		requestData := &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    requestJSON,
			Headers: headers,
		}
		requests = append(requests, requestData)
	}

	return requests, errors
}

func (a adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidReq openrtb2.BidRequest
	if err := json.Unmarshal(requestData.Body, &bidReq); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidderResponse := adapters.NewBidderResponse()
	bidderResponse.Currency = "USD"

	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			bidType := openrtb_ext.BidTypeBanner
			if bidReq.Imp[0].Video != nil {
				bidType = openrtb_ext.BidTypeVideo
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				BidType: bidType,
				Bid:     &seatBid.Bid[i],
			})
		}
	}

	return bidderResponse, nil
}
