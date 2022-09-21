package personaly

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
var personalySKADNetIDs = map[string]bool{
	"44jx6755aq.skadnetwork": true,
}

type adapter struct {
	endpoint string
}

func (a *adapter) Name() string {
	return "personaly"
}

type personalyVideoExt struct {
	PlacementType string `json:"placementtype"`
	Orientation   string `json:"orientation"`
	Skip          int    `json:"skip"`
	SkipDelay     int    `json:"skipdelay"`
}

type personalyBannerExt struct {
	PlacementType           string `json:"placementtype"`
	AllowsCustomCloseButton bool   `json:"allowscustomclosebutton"`
}

type personalyImpExt struct {
	Rewarded int                `json:"rewarded"`
	SKADN    *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type personalyAppExt struct {
	AppStoreID string `json:"appstoreid"`
}

type reqSourceExt struct {
	HeaderBidding int `json:"header_bidding,omitempty"`
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
	personalyRequest := *request

	// Updating app extension
	if personalyRequest.App != nil {

		// *personalyRequest.App creates a copy of the object in appCopy -> correct way.
		// if we do personalyRequest.App just copies the reference -> Not the correct way because
		// if any of the nested property is changed it change others references to and leads to
		// change in other DSPs bidder requests as well.
		appCopy := *personalyRequest.App
		appCopy.Ext, err = json.Marshal(personalyAppExt{
			AppStoreID: personalyRequest.App.Bundle,
		})
		if err != nil {
			errs = append(errs, err)
		}
		personalyRequest.App = &appCopy
	}

	requestImpCopy := personalyRequest.Imp

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

		var personalyExt openrtb_ext.ExtImpTJXPersonaly
		if err = json.Unmarshal(bidderExt.Bidder, &personalyExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		personalyRequest.BApp = nil
		personalyRequest.BAdv = nil
		if personalyExt.Blocklist.BApp != nil {
			personalyRequest.BApp = personalyExt.Blocklist.BApp
		}
		if personalyExt.Blocklist.BAdv != nil {
			personalyRequest.BAdv = personalyExt.Blocklist.BAdv
		}

		// default is interstitial
		placementType := adapters.Interstitial
		rewarded := 0
		if personalyExt.Video.Skip == 0 {
			placementType = adapters.Rewarded
			rewarded = 1
		}

		if thisImp.Video != nil {
			orientation := Horizontal
			if personalyExt.Video.Width < personalyExt.Video.Height {
				orientation = Vertical
			}

			videoCopy := *thisImp.Video
			videoExt := personalyVideoExt{
				PlacementType: string(placementType),
				Orientation:   string(orientation),
				Skip:          personalyExt.Video.Skip,
				SkipDelay:     personalyExt.Video.SkipDelay,
			}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			if personalyExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner
				bannerExt := personalyBannerExt{
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
		if personalyExt.BidFloor != nil {
			thisImp.BidFloor = *personalyExt.BidFloor
		}

		impExt := personalyImpExt{
			Rewarded: rewarded,
		}
		// Add SKADN if supported and present
		if personalyExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, personalySKADNetIDs)
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

		personalyRequest.Imp = []openrtb.Imp{thisImp}
		personalyRequest.Cur = nil
		personalyRequest.Ext = nil

		reqJSON, err := json.Marshal(personalyRequest)
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
					Supported: personalyExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: personalyExt.MRAIDSupported,
				},
				Blocklist: adapters.DynamicBlocklist{
					BApp: personalyRequest.BApp,
					BAdv: personalyRequest.BAdv,
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
