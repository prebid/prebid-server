package scalemonk

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

// Region ...
type Region string

const (
	USEast Region = "us_east"
)

// Orientation ...
type Orientation string

const (
	Horizontal Orientation = "h"
	Vertical   Orientation = "v"
)

// SKAN IDs must be lower case
var scalemonkSKADNetIDs = map[string]bool{
	"av6w8kgt66.skadnetwork": true,
}

type adapter struct {
	endpoint         string
	SupportedRegions map[Region]string
}

type scalemonkVideoExt struct {
	PlacementType string `json:"placementtype"`
	Orientation   string `json:"orientation"`
	Skip          int    `json:"skip"`
	SkipDelay     int    `json:"skipdelay"`
}

type scalemonkBannerExt struct {
	PlacementType           string `json:"placementtype"`
	AllowsCustomCloseButton bool   `json:"allowscustomclosebutton"`
}

type scalemonkImpExt struct {
	Rewarded int                `json:"rewarded"`
	SKADN    *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type scalemonkAppExt struct {
	AppStoreID string `json:"appstoreid"`
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
		SupportedRegions: map[Region]string{
			USEast: config.XAPI.EndpointUSEast,
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

	// copy the bidder request
	scalemonkRequest := *request

	// Updating app extension
	if scalemonkRequest.App != nil {

		// *scalemonkRequest.App creates a copy of the object in appCopy -> correct way.
		// if we do scalemonkRequest.App just copies the reference -> Not the correct way because
		// if any of the nested property is changed it change others references to and leads to
		// change in other DSPs bidder requests as well.
		appCopy := *scalemonkRequest.App
		appCopy.Ext, err = json.Marshal(scalemonkAppExt{
			AppStoreID: scalemonkRequest.App.Bundle,
		})
		if err != nil {
			errs = append(errs, err)
		}
		scalemonkRequest.App = &appCopy
	}

	requestImpCopy := scalemonkRequest.Imp

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

		var scalemonkExt openrtb_ext.ExtImpTJXScaleMonk
		if err = json.Unmarshal(bidderExt.Bidder, &scalemonkExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// default is interstitial
		placementType := adapters.Interstitial
		rewarded := 0
		if scalemonkExt.Video.Skip == 0 {
			placementType = adapters.Rewarded
			rewarded = 1
		}

		if thisImp.Video != nil {
			orientation := Horizontal
			if scalemonkExt.Video.Width < scalemonkExt.Video.Height {
				orientation = Vertical
			}

			videoCopy := *thisImp.Video

			videoExt := scalemonkVideoExt{
				PlacementType: string(placementType),
				Orientation:   string(orientation),
				Skip:          scalemonkExt.Video.Skip,
				SkipDelay:     scalemonkExt.Video.SkipDelay,
			}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			if scalemonkExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner
				bannerExt := scalemonkBannerExt{
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
		if scalemonkExt.BidFloor != nil {
			thisImp.BidFloor = *scalemonkExt.BidFloor
		}

		impExt := scalemonkImpExt{
			Rewarded: rewarded,
		}
		// Add SKADN if supported and present
		if scalemonkExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, scalemonkSKADNetIDs)
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

		scalemonkRequest.Imp = []openrtb.Imp{thisImp}
		scalemonkRequest.Cur = nil
		scalemonkRequest.Ext = nil

		reqJSON, err := json.Marshal(scalemonkRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		uri := a.endpoint

		if endpoint, ok := a.SupportedRegions[Region(scalemonkExt.Region)]; ok {
			uri = endpoint
		}

		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        "scalemonk",
				PlacementType: placementType,
				Region:        scalemonkExt.Region,
				SKAN: adapters.SKAN{
					Supported: scalemonkExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: scalemonkExt.MRAIDSupported,
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
