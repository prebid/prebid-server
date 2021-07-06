package taurusx

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/config"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/cache/skanidlist"
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
	Rewarded                int  `json:"rewarded"`
	AllowsCustomCloseButton bool `json:"allowscustomclosebutton"`
}

type taurusxImpExt struct {
	SKADN *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type adapter struct {
	http             *adapters.HTTPAdapter
	endpoint         string
	SupportedRegions map[Region]string
}

func (adapter *adapter) Name() string {
	return "taurusx"
}

func (adapter *adapter) SkipNoCookies() bool {
	return false
}

func (adapter *adapter) Call(_ context.Context, _ *pbs.PBSRequest, _ *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	return pbs.PBSBidSlice{}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
		SupportedRegions: map[Region]string{
			USEast: config.XAPI.EndpointUSEast,
			JP:     config.XAPI.EndpointJP,
			SG:     config.XAPI.EndpointSG,
		},
	}
	return bidder, nil
}

func NewTaurusXLegacyAdapter(config *adapters.HTTPAdapterConfig, uri, useast, jp, sg string) *adapter {
	return NewTaurusXBidder(adapters.NewHTTPAdapter(config).Client, uri, useast, jp, sg)
}

func NewTaurusXBidder(client *http.Client, uri, useast, jp, sg string) *adapter {
	return &adapter{
		http:     &adapters.HTTPAdapter{Client: client},
		endpoint: uri,
		SupportedRegions: map[Region]string{
			USEast: useast,
			JP:     jp,
			SG:     sg,
		},
	}
}

// MakeRequests ...
func (adapter *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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
		skanSent := false

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

		if thisImp.Video != nil {
			videoCopy := *thisImp.Video

			videoCopy.Ext, err = json.Marshal(&impVideoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			if taurusxExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner
				bannerExt := taurusxBannerExt{
					Rewarded:                taurusxExt.Reward,
					AllowsCustomCloseButton: false,
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
			skanIDList := skanidlist.Get(openrtb_ext.BidderTaurusX)

			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, skanIDList)

			// only add if present
			if len(skadn.SKADNetIDs) > 0 {
				impExt.SKADN = &skadn
				skanSent = true
			}
		}

		thisImp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// reinit the values in the request object
		request.Imp = []openrtb2.Imp{thisImp}

		// json marshal the request
		reqJSON, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// assign the default uri
		uri := adapter.endpoint

		// assign a region based uri if it exists
		if endpoint, ok := adapter.SupportedRegions[Region(taurusxExt.Region)]; ok {
			uri = endpoint
		}

		// Tapjoy Record placement type
		placementType := adapters.Interstitial
		if taurusxExt.Reward == 1 {
			placementType = adapters.Rewarded
		}

		// build request data object
		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        adapter.Name(),
				PlacementType: placementType,
				Region:        taurusxExt.Region,
				SKAN: adapters.SKAN{
					Supported: taurusxExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: taurusxExt.MRAIDSupported,
				},
			},
		}

		// append to request data array
		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

// MakeBids ...
func (adapter *adapter) MakeBids(_ *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))

	var bidReq openrtb2.BidRequest
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
