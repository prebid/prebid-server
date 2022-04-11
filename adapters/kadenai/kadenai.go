package kadenai

import (
	"bytes"
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

// Orientation ...
type Orientation string

const (
	Horizontal Orientation = "h"
	Vertical   Orientation = "v"
)

// SKAN IDs must be lower case
var kadenaiSKADNetIDs = map[string]bool{
	"x44k69ngh6.skadnetwork": true,
}

type adapter struct {
	http     *adapters.HTTPAdapter
	endpoint string
}

func (a *adapter) Name() string {
	return "kadenai"
}

func (a *adapter) SkipNoCookies() bool {
	return false
}

type kadenaiVideoExt struct {
	PlacementType string `json:"placementtype"`
	Orientation   string `json:"orientation"`
	Skip          int    `json:"skip"`
	SkipDelay     int    `json:"skipdelay"`
}

type kadenaiBannerExt struct {
	PlacementType           string `json:"placementtype"`
	AllowsCustomCloseButton bool   `json:"allowscustomclosebutton"`
}

type kadenaiImpExt struct {
	Rewarded int                `json:"rewarded"`
	SKADN    *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type kadenaiAppExt struct {
	AppStoreID string `json:"appstoreid"`
}

type callOneObject struct {
	requestJson bytes.Buffer
	mediaType   pbs.MediaType
}

func (a *adapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	return pbs.PBSBidSlice{}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func NewKadenAILegacyAdapter(config *adapters.HTTPAdapterConfig, uri string) *adapter {
	return NewKadenAIBidder(adapters.NewHTTPAdapter(config).Client, uri)
}

func NewKadenAIBidder(client *http.Client, uri string) *adapter {
	return &adapter{
		http:     &adapters.HTTPAdapter{Client: client},
		endpoint: uri,
	}
}

// MakeRequests ...
func (a *adapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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
		appExt := kadenaiAppExt{
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

		var kadenaiExt openrtb_ext.ExtImpKadenAI
		if err = json.Unmarshal(bidderExt.Bidder, &kadenaiExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// default is interstitial
		placementType := adapters.Interstitial
		rewarded := 0
		if kadenaiExt.Video.Skip == 0 {
			placementType = adapters.Rewarded
			rewarded = 1
		}

		if thisImp.Video != nil {
			orientation := Horizontal
			if kadenaiExt.Video.Width < kadenaiExt.Video.Height {
				orientation = Vertical
			}

			videoCopy := *thisImp.Video
			videoExt := kadenaiVideoExt{
				PlacementType: string(placementType),
				Orientation:   string(orientation),
				Skip:          kadenaiExt.Video.Skip,
				SkipDelay:     kadenaiExt.Video.SkipDelay,
			}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			if kadenaiExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner
				bannerExt := kadenaiBannerExt{
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

		// Overwrite BidFloor if present
		if kadenaiExt.BidFloor != nil {
			thisImp.BidFloor = *kadenaiExt.BidFloor
		}

		impExt := kadenaiImpExt{
			Rewarded: rewarded,
		}
		// Add SKADN if supported and present
		if kadenaiExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, kadenaiSKADNetIDs)
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

		request.Imp = []openrtb.Imp{thisImp}
		request.Cur = nil
		request.Ext = nil

		reqJSON, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    reqJSON,
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        a.Name(),
				PlacementType: placementType,
				Region:        "us_east",
				SKAN: adapters.SKAN{
					Supported: kadenaiExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: kadenaiExt.MRAIDSupported,
				},
			},
		}

		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

// MakeBids ...
func (a *adapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
