package mintegral

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/config"

	openrtb "github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
)

// Region ...
type Region string

const (
	HK Region = "hk"
	SG Region = "sg"
	VG Region = "vg"
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
	http             *adapters.HTTPAdapter
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
	PlacementType string `json:"placementtype"`
	Orientation   int    `json:"orientation"`
	Skip          int    `json:"skip"`
	SkipDelay     int    `json:"skipdelay"`
}

type mintegralBannerExt struct {
	PlacementType           string `json:"placementtype"`
	AllowsCustomCloseButton bool   `json:"allowscustomclosebutton"`
}

type mintegralImpExt struct {
	Rewarded int                `json:"rewarded"`
	SKADN    *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type mintegralAppExt struct {
	AppStoreID string `json:"appstoreid"`
}

func (a *adapter) Call(_ context.Context, _ *pbs.PBSRequest, _ *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	return pbs.PBSBidSlice{}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
		SupportedRegions: map[Region]string{
			HK: config.XAPI.EndpointHK,
			SG: config.XAPI.EndpointSG,
			VG: config.XAPI.EndpointVG,
		},
	}
	return bidder, nil
}

func NewMintegralLegacyAdapter(config *adapters.HTTPAdapterConfig, uri string, hk, sg, vg string) *adapter {
	return NewMintegralBidder(adapters.NewHTTPAdapter(config).Client, uri, hk, sg, vg)
}

func NewMintegralBidder(client *http.Client, uri string, hk, sg, vg string) *adapter {
	return &adapter{
		http:     &adapters.HTTPAdapter{Client: client},
		endpoint: uri,
		SupportedRegions: map[Region]string{
			HK: hk,
			SG: sg,
			VG: vg,
		},
	}
}

// MakeRequests ...
func (a *adapter) MakeRequests(request *openrtb.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)

	requestData := make([]*adapters.RequestData, 0, numRequests)
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	errs := make([]error, 0, len(request.Imp))
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

	requestImpCopy := request.Imp

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

		var mintegralExt openrtb_ext.ExtImpMintegral
		if err = json.Unmarshal(bidderExt.Bidder, &mintegralExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// default is interstitial
		placementType := adapters.Interstitial
		rewarded := 0
		if mintegralExt.Video.Skip == 0 {
			placementType = adapters.Rewarded
			rewarded = 1
		}

		if thisImp.Video != nil {
			orientation := Horizontal
			if mintegralExt.Video.Width < mintegralExt.Video.Height {
				orientation = Vertical
			}

			videoCopy := *thisImp.Video
			videoExt := mintegralVideoExt{
				PlacementType: string(placementType),
				Orientation:   int(orientation),
				Skip:          mintegralExt.Video.Skip,
				SkipDelay:     mintegralExt.Video.SkipDelay,
			}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			if mintegralExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner
				bannerExt := mintegralBannerExt{
					PlacementType:           string(placementType),
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

		impExt := mintegralImpExt{
			Rewarded: rewarded,
		}
		// Add SKADN if supported and present
		if mintegralExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, mintegralSKADNetIDs)
			if len(skadn.SKADNetIDs) > 0 {
				skanSent = true
				impExt.SKADN = &skadn
			}
		}

		thisImp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		reqSourceExt := map[string]interface{}{}
		reqSourceExt["mediator_id"] = mintegralExt.MediatorID
		reqSourceExt["header_bidding"] = mintegralExt.HeaderBidding

		if request.Source != nil {
			reqSource := *request.Source

			if reqSource.Ext != nil {
				if err = json.Unmarshal(reqSource.Ext, &reqSourceExt); err != nil {
					errs = append(errs, &errortypes.BadInput{
						Message: err.Error(),
					})
					continue
				}
			}

			reqSource.Ext, err = json.Marshal(&reqSourceExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			request.Source = &reqSource
		} else {
			reqSource := openrtb.Source{}

			reqSource.Ext, err = json.Marshal(&reqSourceExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			request.Source = &reqSource
		}

		request.Imp = []openrtb.Imp{thisImp}
		request.Cur = nil
		request.Ext = nil

		reqJSON, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		uri := a.endpoint

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
			},
		}

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
