package kadenai

import (
	"encoding/json"
	"net/http"

	"github.com/prebid/prebid-server/config"

	openrtb "github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/tjx_base"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
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
	endpoint string
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

type reqSourceExt struct {
	HeaderBidding int `json:"header_bidding,omitempty"`
}

type kadenaiAppExt struct {
	AppStoreID string `json:"appstoreid"`
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
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

	// copy the bidder request
	kadenaiRequest := *request

	// Updating app extension
	if kadenaiRequest.App != nil {

		// *kadenaiRequest.App creates a copy of the object in appCopy -> correct way.
		// if we do kadenaiRequest.App just copies the reference -> Not the correct way because
		// if any of the nested property is changed it change others references to and leads to
		// change in other DSPs bidder requests as well.
		appCopy := *kadenaiRequest.App
		appCopy.Ext, err = json.Marshal(kadenaiAppExt{
			AppStoreID: kadenaiRequest.App.Bundle,
		})
		if err != nil {
			errs = append(errs, err)
		}
		kadenaiRequest.App = &appCopy
	}

	requestImpCopy := kadenaiRequest.Imp

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

		var kadenaiExt openrtb_ext.ExtImpTJXKadenAI
		if err = json.Unmarshal(bidderExt.Bidder, &kadenaiExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		kadenaiRequest.BApp = nil
		kadenaiRequest.BAdv = nil
		if kadenaiExt.Blocklist.BApp != nil {
			kadenaiRequest.BApp = kadenaiExt.Blocklist.BApp
		}
		if kadenaiExt.Blocklist.BAdv != nil {
			kadenaiRequest.BAdv = kadenaiExt.Blocklist.BAdv
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

		kadenaiRequest.Imp = []openrtb.Imp{thisImp}
		kadenaiRequest.Cur = nil
		kadenaiRequest.Ext = nil

		reqJSON, err := json.Marshal(kadenaiRequest)
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
				Bidder:        "kadenai",
				PlacementType: placementType,
				Region:        "us_east",
				SKAN: adapters.SKAN{
					Supported: kadenaiExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: kadenaiExt.MRAIDSupported,
				},
				Blocklist: adapters.DynamicBlocklist{
					BApp: kadenaiRequest.BApp,
					BAdv: kadenaiRequest.BAdv,
				},
			},
		}

		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

// MakeBids ...
func (a *adapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return tjx_base.MakeBids(internalRequest, externalRequest, response)
}
