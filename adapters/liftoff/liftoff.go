package liftoff

import (
	"bytes"
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

// Orientation ...
type Orientation string

const (
	Horizontal Orientation = "h"
	Vertical   Orientation = "v"
)

// SKAN IDs must be lower case
var liftoffSKADNetIDs = map[string]bool{
	"7ug5zh24hu.skadnetwork": true,
}

// LiftoffAdapter ...
type LiftoffAdapter struct {
	http             *adapters.HTTPAdapter
	URI              string
	SupportedRegions map[Region]string
}

// Name is used for cookies and such
func (a *LiftoffAdapter) Name() string {
	return "liftoff"
}

// SkipNoCookies ...
func (a *LiftoffAdapter) SkipNoCookies() bool {
	return false
}

type liftoffVideoExt struct {
	PlacementType string `json:"placementtype"`
	Orientation   string `json:"orientation"`
	Skip          int    `json:"skip"`
	SkipDelay     int    `json:"skipdelay"`
}

type liftoffBannerExt struct {
	PlacementType           string `json:"placementtype"`
	AllowsCustomCloseButton bool   `json:"allowscustomclosebutton"`
}

type liftoffImpExt struct {
	Rewarded int                `json:"rewarded"`
	SKADN    *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type liftoffAppExt struct {
	AppStoreID string `json:"appstoreid"`
}

type callOneObject struct {
	requestJson bytes.Buffer
	mediaType   pbs.MediaType
}

// Call is legacy, and added only to support LiftoffAdapter interface
func (a *LiftoffAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	return pbs.PBSBidSlice{}, nil
}

// NewLiftoffAdapter ...
func NewLiftoffAdapter(config *adapters.HTTPAdapterConfig, uri string, useast string, eu string, apac string) *LiftoffAdapter {
	return NewLiftoffBidder(adapters.NewHTTPAdapter(config).Client, uri, useast, eu, apac)
}

// NewLiftoffBidder ...
func NewLiftoffBidder(client *http.Client, uri string, useast string, eu string, apac string) *LiftoffAdapter {
	a := &adapters.HTTPAdapter{Client: client}

	return &LiftoffAdapter{
		http: a,
		URI:  uri,
		SupportedRegions: map[Region]string{
			USEast: useast,
			EU:     eu,
			APAC:   apac,
		},
	}
}

// MakeRequests ...
func (a *LiftoffAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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
		appExt := liftoffAppExt{
			AppStoreID: request.App.Bundle,
		}
		request.App.Ext, err = json.Marshal(&appExt)
		if err != nil {
			errs = append(errs, err)
		}
	}

	requestImpCopy := request.Imp

	for i := 0; i < numRequests; i++ {
		thisImp := requestImpCopy[i]

		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(thisImp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		var liftoffExt openrtb_ext.ExtImpLiftoff
		if err = json.Unmarshal(bidderExt.Bidder, &liftoffExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// default is interstitial
		placementType := adapters.Interstitial
		rewarded := 0
		if liftoffExt.Video.Skip == 0 {
			placementType = adapters.Rewarded
			rewarded = 1
		}

		if thisImp.Video != nil {
			orientation := Horizontal
			if liftoffExt.Video.Width < liftoffExt.Video.Height {
				orientation = Vertical
			}

			videoCopy := *thisImp.Video
			videoExt := liftoffVideoExt{
				PlacementType: string(placementType),
				Orientation:   string(orientation),
				Skip:          liftoffExt.Video.Skip,
				SkipDelay:     liftoffExt.Video.SkipDelay,
			}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			if liftoffExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner
				bannerExt := liftoffBannerExt{
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

		impExt := liftoffImpExt{
			Rewarded: rewarded,
		}
		// Add SKADN if supported and present
		if liftoffExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, liftoffSKADNetIDs)
			if len(skadn.SKADNetIDs) > 0 {
				impExt.SKADN = &skadn
			}
		}

		thisImp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		request.Imp = []openrtb.Imp{thisImp}
		request.Cur = nil
		request.Ext = nil

		reqJSON, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		uri := a.URI
		if endpoint, ok := a.SupportedRegions[Region(liftoffExt.Region)]; ok {
			uri = endpoint
		}

		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,
		}

		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

// MakeBids ...
func (a *LiftoffAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
