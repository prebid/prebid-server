package taurusx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
)

// Region ...
type Region string

const (
	USEast Region = "us_east"
	JP     Region = "jp"
	SG     Region = "sg"
)

// SKAN IDs must be lower case
var taurusxExtSKADNetIDs = map[string]bool{
	"22mmun2rn5.skadnetwork": true,
}

type taurusxVideoExt struct {
	Rewarded int `json:"rewarded"`
}

type taurusxBannerExt struct {
	Rewarded int `json:"rewarded"`
}

type taurusxImpExt struct {
	SKADN openrtb_ext.SKADN `json:"skadn,omitempty"`
}

// TaurusXAdapter ...
type TaurusXAdapter struct {
	http             *adapters.HTTPAdapter
	URI              string
	SupportedRegions map[Region]string
}

// Name is used for cookies and such
func (adapter *TaurusXAdapter) Name() string {
	return "taurusx"
}

// SkipNoCookies ...
func (adapter *TaurusXAdapter) SkipNoCookies() bool {
	return false
}

// Call is legacy, and added only to support TaurusXAdapter interface
func (adapter *TaurusXAdapter) Call(_ context.Context, _ *pbs.PBSRequest, _ *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	return pbs.PBSBidSlice{}, nil
}

// NewTaurusXAdapter ...
func NewTaurusXAdapter(config *adapters.HTTPAdapterConfig, uri, useast, jp, sg string) *TaurusXAdapter {
	return NewTaurusXBidder(adapters.NewHTTPAdapter(config).Client, uri, useast, jp, sg)
}

// NewTaurusXBidder ...
func NewTaurusXBidder(client *http.Client, uri, useast, jp, sg string) *TaurusXAdapter {
	adapter := &adapters.HTTPAdapter{Client: client}

	return &TaurusXAdapter{
		http: adapter,
		URI:  uri,
		SupportedRegions: map[Region]string{
			USEast: useast,
			JP:     jp,
			SG:     sg,
		},
	}
}

// MakeRequests ...
func (adapter *TaurusXAdapter) MakeRequests(request *openrtb.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)

	requestData := make([]*adapters.RequestData, 0, numRequests)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	errs := make([]error, 0, numRequests)

	// clone the request imp array
	requestImpCopy := request.Imp

	var err error

	for i := 0; i < numRequests; i++ {
		// clone current imp
		thisImp := requestImpCopy[i]

		// extract bidder extension
		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(thisImp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// unmarshal bidder extension to taurusx extension
		var taurusxExt openrtb_ext.ExtImpTaurusX
		if err = json.Unmarshal(bidderExt.Bidder, &taurusxExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		impVideoExt := taurusxVideoExt{
			Rewarded: taurusxExt.Reward,
		}

		thisImp.Video.Ext, err = json.Marshal(&impVideoExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if thisImp.Banner != nil {
			if taurusxExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner

				bannerExt := taurusxBannerExt{
					Rewarded: taurusxExt.Reward,
				}
				bannerCopy.Ext, err = json.Marshal(&bannerExt)
				if err != nil {
					errs = append(errs, err)
					continue
				}

				thisImp.Banner = &bannerCopy
			} else {
				thisImp.Banner = nil
			}
		}

		impExt := taurusxImpExt{}
		if taurusxExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, taurusxExtSKADNetIDs)
			// only add if present
			if len(skadn.SKADNetIDs) > 0 {
				impExt.SKADN = skadn
			}
		}

		thisImp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// reinit the values in the request object
		request.Imp = []openrtb.Imp{thisImp}

		// json marshal the request
		reqJSON, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// assign the default uri
		uri := adapter.URI

		// assign a region based uri if it exists
		if endpoint, ok := adapter.SupportedRegions[Region(taurusxExt.Region)]; ok {
			uri = endpoint
		}

		// build request data object
		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,
		}

		// append to request data array
		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

// MakeBids ...
func (adapter *TaurusXAdapter) MakeBids(_ *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))

	var bidReq openrtb.BidRequest
	if err := json.Unmarshal(externalRequest.Body, &bidReq); err != nil {
		return nil, []error{err}
	}

	bidType := openrtb_ext.BidTypeBanner

	if bidReq.Imp[0].Video != nil {
		bidType = openrtb_ext.BidTypeVideo
	}

	for _, sb := range bidResp.SeatBid {
		for _, b := range sb.Bid {
			if b.Price != 0 {
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &b,
					BidType: bidType,
				})
			}
		}
	}

	return bidResponse, nil
}
