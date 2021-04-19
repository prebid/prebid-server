package molococloud

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
	EU     Region = "eu"
	APAC   Region = "apac"
)

// SKAN IDs must be lower case
var molocoCloudSKADNetIDs = map[string]bool{
	"9t245vhmpl.skadnetwork": true,
}

type molocoCloudVideoExt struct {
	PlacementType adapters.PlacementType `json:"placementtype"`
}

type molocoCloudBannerExt struct {
	PlacementType           adapters.PlacementType `json:"placementtype"`
	AllowsCustomCloseButton bool                   `json:"allowscustomclosebutton"`
}

type molocoCloudImpExt struct {
	SKADN *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

// MolocoCloudAdapter ...
type MolocoCloudAdapter struct {
	http             *adapters.HTTPAdapter
	URI              string
	SupportedRegions map[Region]string
}

// Name is used for cookies and such
func (adapter *MolocoCloudAdapter) Name() string {
	return "molococloud"
}

// SkipNoCookies ...
func (adapter *MolocoCloudAdapter) SkipNoCookies() bool {
	return false
}

// Call is legacy, and added only to support MolocoCloudAdapter interface
func (adapter *MolocoCloudAdapter) Call(_ context.Context, _ *pbs.PBSRequest, _ *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	return pbs.PBSBidSlice{}, nil
}

// NewMolocoCloudAdapter ...
func NewMolocoCloudAdapter(config *adapters.HTTPAdapterConfig, uri, useast, eu, apac string) *MolocoCloudAdapter {
	return NewMolocoCloudBidder(adapters.NewHTTPAdapter(config).Client, uri, useast, eu, apac)
}

// NewMolocoCloudBidder ...
func NewMolocoCloudBidder(client *http.Client, uri, useast, eu, apac string) *MolocoCloudAdapter {
	adapter := &adapters.HTTPAdapter{Client: client}

	return &MolocoCloudAdapter{
		http: adapter,
		URI:  uri,
		SupportedRegions: map[Region]string{
			USEast: useast,
			EU:     eu,
			APAC:   apac,
		},
	}
}

// MakeRequests ...
func (adapter *MolocoCloudAdapter) MakeRequests(request *openrtb.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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

		// unmarshal bidder extension to moloco cloud extension
		var molocoCloudExt openrtb_ext.ExtImpMolocoCloud
		if err = json.Unmarshal(bidderExt.Bidder, &molocoCloudExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// placement type is either Rewarded or Interstitial, default is Interstitial
		placementType := adapters.Interstitial
		if molocoCloudExt.PlacementType == string(adapters.Rewarded) {
			placementType = adapters.Rewarded
		}

		if thisImp.Video != nil {
			// instantiate moloco cloud video extension struct
			videoExt := molocoCloudVideoExt{
				PlacementType: placementType,
			}

			// clone the current video element
			videoCopy := *thisImp.Video

			// assign moloco cloud video extension to cloned video element
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			// assign cloned video element to imp object
			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			if molocoCloudExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner

				bannerExt := molocoCloudBannerExt{
					PlacementType:           placementType,
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

		// Add impression extensions
		impExt := molocoCloudImpExt{}

		// Add SKADN if supported and present=
		if molocoCloudExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, molocoCloudSKADNetIDs)
			if len(skadn.SKADNetIDs) > 0 {
				impExt.SKADN = &skadn
			}
		}

		thisImp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// reinit the values in the request object
		request.Imp = []openrtb.Imp{thisImp}
		request.Cur = nil
		request.Ext = nil

		// json marshal the request
		reqJSON, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// assign the default uri
		uri := adapter.URI

		// assign a region based uri if it exists
		if endpoint, ok := adapter.SupportedRegions[Region(molocoCloudExt.Region)]; ok {
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
func (adapter *MolocoCloudAdapter) MakeBids(_ *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
				// copy response.bidid or response.id to openrtb_response.seatbid.bid.bidid
				if b.ID == "1" {
					if len(bidResp.BidID) > 0 {
						b.ID = bidResp.BidID
					} else if len(bidResp.ID) > 0 {
						b.ID = bidResp.ID
					}
				}

				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &b,
					BidType: bidType,
				})
			}
		}
	}

	return bidResponse, nil
}
