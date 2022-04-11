package appier

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
	USEast Region = "us_east"
	EMEA   Region = "emea"
	JP     Region = "jp"
	SG     Region = "sg"
)

// Orientation ...
type Orientation int

const (
	Vertical   Orientation = 1
	Horizontal Orientation = 2
)

// SKAN IDs must be lower case
var appierSKADNetIDs = map[string]bool{
	"v72qych5uu.skadnetwork": true,
}

type adapter struct {
	http             *adapters.HTTPAdapter
	endpoint         string
	SupportedRegions map[Region]string
}

func (a *adapter) Name() string {
	return "appier"
}

func (a *adapter) SkipNoCookies() bool {
	return false
}

type appierVideoExt struct {
	PlacementType string `json:"placementtype"`
	Orientation   int    `json:"orientation"`
	Skip          int    `json:"skip"`
	SkipDelay     int    `json:"skipdelay"`
}

type appierBannerExt struct {
	PlacementType           string `json:"placementtype"`
	AllowsCustomCloseButton bool   `json:"allowscustomclosebutton"`
}

type appierImpExt struct {
	Rewarded int                `json:"rewarded"`
	SKADN    *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type appierAppExt struct {
	AppStoreID string `json:"appstoreid"`
}

func (a *adapter) Call(_ context.Context, _ *pbs.PBSRequest, _ *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	return pbs.PBSBidSlice{}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
		SupportedRegions: map[Region]string{
			USEast: config.XAPI.EndpointUSEast,
			EMEA:   config.XAPI.EndpointEMEA,
			JP:     config.XAPI.EndpointJP,
			SG:     config.XAPI.EndpointSG,
		},
	}
	return bidder, nil
}

func NewAppierLegacyAdapter(config *adapters.HTTPAdapterConfig, uri, useast, emea, jp, sg string) *adapter {
	return NewAppierBidder(adapters.NewHTTPAdapter(config).Client, uri, useast, emea, jp, sg)
}

func NewAppierBidder(client *http.Client, uri, useast, emea, jp, sg string) *adapter {
	return &adapter{
		http:     &adapters.HTTPAdapter{Client: client},
		endpoint: uri,
		SupportedRegions: map[Region]string{
			USEast: useast,
			EMEA:   emea,
			JP:     jp,
			SG:     sg,
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
		appExt := appierAppExt{
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

		var appierExt openrtb_ext.ExtImpAppier
		if err = json.Unmarshal(bidderExt.Bidder, &appierExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// default is interstitial
		placementType := adapters.Interstitial
		rewarded := 0
		if appierExt.Video.Skip == 0 {
			placementType = adapters.Rewarded
			rewarded = 1
		}

		if thisImp.Video != nil {
			orientation := Horizontal
			if appierExt.Video.Width < appierExt.Video.Height {
				orientation = Vertical
			}

			videoCopy := *thisImp.Video

			if appierExt.EndcardHTMLSupported {
				videoCopy.CompanionType = append(videoCopy.CompanionType, openrtb.CompanionTypeHTML)
			}

			videoExt := appierVideoExt{
				PlacementType: string(placementType),
				Orientation:   int(orientation),
				Skip:          appierExt.Video.Skip,
				SkipDelay:     appierExt.Video.SkipDelay,
			}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			if appierExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner
				bannerExt := appierBannerExt{
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
		if appierExt.BidFloor != nil {
			thisImp.BidFloor = *appierExt.BidFloor
		}

		impExt := appierImpExt{
			Rewarded: rewarded,
		}
		// Add SKADN if supported and present
		if appierExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, appierSKADNetIDs)
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

		uri := a.endpoint

		if endpoint, ok := a.SupportedRegions[Region(appierExt.Region)]; ok {
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
				Region:        appierExt.Region,
				SKAN: adapters.SKAN{
					Supported: appierExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: appierExt.MRAIDSupported,
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
