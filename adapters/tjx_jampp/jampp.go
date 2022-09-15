package jampp

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
var jamppSKADNetIDs = map[string]bool{
	"yclnxrl5pm.skadnetwork": true,
}

type adapter struct {
	endpoint         string
	SupportedRegions map[Region]string
}

type jamppVideoExt struct {
	PlacementType string `json:"placementtype"`
	Orientation   string `json:"orientation"`
	Skip          int    `json:"skip"`
	SkipDelay     int    `json:"skipdelay"`
}

type jamppBannerExt struct {
	PlacementType           string `json:"placementtype"`
	AllowsCustomCloseButton bool   `json:"allowscustomclosebutton"`
}

type jamppImpExt struct {
	Rewarded int                `json:"rewarded"`
	SKADN    *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type reqSourceExt struct {
	HeaderBidding int `json:"header_bidding,omitempty"`
}

type jamppAppExt struct {
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
func (a *adapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)

	requestData := make([]*adapters.RequestData, 0, numRequests)
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	errs := make([]error, 0, len(request.Imp))
	var err error

	// copy the bidder request
	jamppRequest := *request

	// Updating app extension
	if jamppRequest.App != nil {

		// *jamppRequest.App creates a copy of the object in appCopy -> correct way.
		// if we do spotadRequest.App just copies the reference -> Not the correct way because
		// if any of the nested property is changed it change others references to and leads to
		// change in other DSPs bidder requests as well.
		appCopy := *jamppRequest.App
		appCopy.Ext, err = json.Marshal(jamppAppExt{
			AppStoreID: jamppRequest.App.Bundle,
		})
		if err != nil {
			errs = append(errs, err)
		}
		jamppRequest.App = &appCopy
	}

	requestImpCopy := jamppRequest.Imp

	var srcExt *reqSourceExt
	if request.Source != nil && request.Source.Ext != nil {
		if err := json.Unmarshal(request.Source.Ext, &srcExt); err != nil {
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

		var jamppExt openrtb_ext.ExtImpTJXJampp
		if err = json.Unmarshal(bidderExt.Bidder, &jamppExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}
		// This check is for identifying if the request comes from TJX
		if srcExt != nil && srcExt.HeaderBidding == 1 {
			jamppRequest.BApp = nil
			jamppRequest.BAdv = nil

			if jamppExt.Blocklist.BApp != nil {
				jamppRequest.BApp = jamppExt.Blocklist.BApp
			}
			if jamppExt.Blocklist.BAdv != nil {
				jamppRequest.BAdv = jamppExt.Blocklist.BAdv
			}
		}
		// default is interstitial
		placementType := adapters.Interstitial
		rewarded := 0
		if jamppExt.Video.Skip == 0 {
			placementType = adapters.Rewarded
			rewarded = 1
		}

		if thisImp.Video != nil {
			orientation := Horizontal
			if jamppExt.Video.Width < jamppExt.Video.Height {
				orientation = Vertical
			}

			videoCopy := *thisImp.Video

			videoExt := jamppVideoExt{
				PlacementType: string(placementType),
				Orientation:   string(orientation),
				Skip:          jamppExt.Video.Skip,
				SkipDelay:     jamppExt.Video.SkipDelay,
			}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			if jamppExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner
				bannerExt := jamppBannerExt{
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
		if jamppExt.BidFloor != nil {
			thisImp.BidFloor = *jamppExt.BidFloor
		}

		impExt := jamppImpExt{
			Rewarded: rewarded,
		}
		// Add SKADN if supported and present
		if jamppExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, jamppSKADNetIDs)
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

		jamppRequest.Imp = []openrtb.Imp{thisImp}
		jamppRequest.Cur = nil
		jamppRequest.Ext = nil

		reqJSON, err := json.Marshal(jamppRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		uri := a.endpoint

		if endpoint, ok := a.SupportedRegions[Region(jamppExt.Region)]; ok {
			uri = endpoint
		}

		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        "jampp",
				PlacementType: placementType,
				Region:        jamppExt.Region,
				SKAN: adapters.SKAN{
					Supported: jamppExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: jamppExt.MRAIDSupported,
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
