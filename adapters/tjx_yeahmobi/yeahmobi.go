package yeahmobi

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
	USEast Region = "us_east"
	EU     Region = "eu"
	APAC   Region = "apac"
)

// Orientation ...
type Orientation string

type yeahmobiImpExt struct {
	Rewarded int                `json:"rewarded"`
	SKADN    *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

// SKAN IDs must be lower case
var yeahmobiSKADNetIDs = map[string]bool{
	"32z4fx6l9h.skadnetwork": true,
}

type adapter struct {
	endpoint         string
	SupportedRegions map[Region]string
}

func (a *adapter) Name() string {
	return "yeahmobi"
}

type yeahmobiVideoExt struct {
	Rewarded int `json:"rewarded"`
}

type yeahmobiUserExt struct {
	TagID    string `json:"tagid"`
	Region   string `json:"region"`
	Language string `json:"language"`
}

type yeahmobiBannerExt struct {
	HasPrivate  int    `json:"hasPrivate"`
	AllowAdType string `json:"allow_ad_type"`
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
		SupportedRegions: map[Region]string{
			USEast: config.XAPI.EndpointUSEast,
			EU:     config.XAPI.EndpointEU,
			APAC:   config.XAPI.EndpointAPAC,
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

	errs := make([]error, 0, len(request.Imp))
	var err error

	yeahmobiRequest := *request

	// yeahmobi does not have a app extension

	requestImpCopy := yeahmobiRequest.Imp

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

		var yeahmobiExt openrtb_ext.ExtImpTJXYeahMobi
		if err = json.Unmarshal(bidderExt.Bidder, &yeahmobiExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		yeahmobiRequest.BApp = nil
		yeahmobiRequest.BAdv = nil
		if yeahmobiExt.Blocklist.BApp != nil {
			yeahmobiRequest.BApp = yeahmobiExt.Blocklist.BApp
		}
		if yeahmobiExt.Blocklist.BAdv != nil {
			yeahmobiRequest.BAdv = yeahmobiExt.Blocklist.BAdv
		}

		// default is interstitial
		placementType := adapters.Interstitial
		rewarded := 0
		if thisImp.Video != nil && *thisImp.Video.Skip == 0 {
			placementType = adapters.Rewarded
			rewarded = 1
		}

		if thisImp.Video != nil {
			videoCopy := *thisImp.Video

			videoExt := yeahmobiVideoExt{
				Rewarded: rewarded,
			}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			bannerCopy := *thisImp.Banner

			// Get banner exts
			var bannerExt yeahmobiBannerExt
			if err = json.Unmarshal(bannerCopy.Ext, &bannerExt); err != nil {
				errs = append(errs, &errortypes.BadInput{
					Message: err.Error(),
				})
				continue
			}

			bannerCopy.Ext, err = json.Marshal(&bannerExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			thisImp.Banner = &bannerCopy
		}

		// Overwrite BidFloor if present
		if yeahmobiExt.BidFloor != nil {
			thisImp.BidFloor = *yeahmobiExt.BidFloor
		}

		// Add impression extensions
		impExt := yeahmobiImpExt{
			Rewarded: rewarded,
		}

		// Add SKADN if supported and present
		if yeahmobiExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, yeahmobiSKADNetIDs)
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

		yeahmobiRequest.Imp = []openrtb.Imp{thisImp}
		yeahmobiRequest.Cur = nil
		yeahmobiRequest.Ext = nil

		reqJSON, err := json.Marshal(yeahmobiRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		uri := a.endpoint

		if endpoint, ok := a.SupportedRegions[Region(yeahmobiExt.Region)]; ok {
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
				Region:        yeahmobiExt.Region,
				SKAN: adapters.SKAN{
					Supported: yeahmobiExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: yeahmobiExt.MRAIDSupported,
				},
				Blocklist: adapters.DynamicBlocklist{
					BApp: yeahmobiRequest.BApp,
					BAdv: yeahmobiRequest.BAdv,
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
