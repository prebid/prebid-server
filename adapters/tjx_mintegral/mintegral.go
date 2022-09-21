package mintegral

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/config"

	openrtb "github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Region ...
type Region string

const (
	SG Region = "sg"
)

// Orientation ...
type Orientation int

const (
	Vertical   Orientation = 1
	Horizontal Orientation = 2
)

// SKAN IDs must be lower case
var mintegralSKADNetIDs = map[string]bool{
	"kbd757ywx3.skadnetwork": true,
}

type adapter struct {
	endpoint         string
	SupportedRegions map[Region]string
}

func (a *adapter) Name() string {
	return "mintegral"
}

func (a *adapter) SkipNoCookies() bool {
	return false
}

type mintegralVideoExt struct {
	PlacementType adapters.PlacementType `json:"placementtype"`
	Orientation   int                    `json:"orientation"`
	Skip          int                    `json:"skip"`
	SkipDelay     int                    `json:"skipdelay"`
}

type mintegralBannerExt struct {
	PlacementType           adapters.PlacementType `json:"placementtype"`
	AllowsCustomCloseButton bool                   `json:"allowscustomclosebutton"`
}

type mintegralImpExt struct {
	Rewarded int                `json:"rewarded"`
	SKADN    *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type mintegralAppExt struct {
	AppStoreID string `json:"appstoreid"`
}

type reqSourceExt struct {
	HeaderBidding int `json:"header_bidding,omitempty"`
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
		SupportedRegions: map[Region]string{
			SG: config.XAPI.EndpointSG,
		},
	}
	return bidder, nil
}

// MakeRequests ...
func (a *adapter) MakeRequests(request *openrtb.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)

	requestData := make([]*adapters.RequestData, 0, numRequests)
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	errs := make([]error, 0, numRequests)

	// copy the bidder request
	mintegralRequest := *request

	// clone the request imp array
	requestImpCopy := mintegralRequest.Imp

	var err error

	// Updating app extension
	if request.App != nil {
		appExt := mintegralAppExt{
			AppStoreID: request.App.Bundle,
		}
		request.App.Ext, err = json.Marshal(&appExt)
		if err != nil {
			errs = append(errs, err)
		}
	}

	for i := 0; i < numRequests; i++ {
		skanSent := false

		thisImp := requestImpCopy[i]

		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(thisImp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		var mintegralExt openrtb_ext.ExtImpTJXMintegral
		if err = json.Unmarshal(bidderExt.Bidder, &mintegralExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		mintegralRequest.BApp = nil
		mintegralRequest.BAdv = nil
		if mintegralExt.Blocklist.BApp != nil {
			mintegralRequest.BApp = mintegralExt.Blocklist.BApp
		}
		if mintegralExt.Blocklist.BAdv != nil {
			mintegralRequest.BAdv = mintegralExt.Blocklist.BAdv
		}

		// default is interstitial
		placementType := adapters.Interstitial
		rewarded := 0
		if mintegralExt.Video.Skip == 0 {
			placementType = adapters.Rewarded
			rewarded = 1
		}

		if thisImp.Video != nil {
			// clone the current video element
			videoCopy := *thisImp.Video

			orientation := Horizontal
			if mintegralExt.Video.Width < mintegralExt.Video.Height {
				orientation = Vertical
			}

			// instantiate mintegral video extension struct
			videoExt := mintegralVideoExt{
				PlacementType: placementType,
				Orientation:   int(orientation),
				Skip:          mintegralExt.Video.Skip,
				SkipDelay:     mintegralExt.Video.SkipDelay,
			}

			// assign mintegral video extension to cloned video element
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			// assign cloned video element to imp object
			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			if mintegralExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner

				bannerExt := mintegralBannerExt{
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

		// Overwrite BidFloor if present
		if mintegralExt.BidFloor != nil {
			thisImp.BidFloor = *mintegralExt.BidFloor
		}

		// Add impression extensions
		impExt := mintegralImpExt{
			Rewarded: rewarded,
		}

		// Add SKADN if supported and present
		if mintegralExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, mintegralSKADNetIDs)
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
		mintegralRequest.Imp = []openrtb.Imp{thisImp}
		mintegralRequest.Cur = nil
		mintegralRequest.Ext = nil

		reqJSON, err := json.Marshal(mintegralRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// assign the default uri
		uri := a.endpoint

		// assign a region based uri if it exists
		if endpoint, ok := a.SupportedRegions[Region(mintegralExt.Region)]; ok {
			uri = endpoint
		}

		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        a.Name(),
				PlacementType: placementType,
				Region:        mintegralExt.Region,
				SKAN: adapters.SKAN{
					Supported: mintegralExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: mintegralExt.MRAIDSupported,
				},
				Blocklist: adapters.DynamicBlocklist{
					BApp: mintegralRequest.BApp,
					BAdv: mintegralRequest.BAdv,
				},
			},
		}

		// append to request data array
		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

// MakeBids ...
func (a *adapter) MakeBids(_ *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
				// copy response.bidid to openrtb_response.seatbid.bid.bidid
				if b.ID == "0" {
					b.ID = bidResp.BidID
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
