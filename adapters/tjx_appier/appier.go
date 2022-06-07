package appier

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
	endpoint         string
	SupportedRegions map[Region]string
}

func (a *adapter) Name() string {
	return "appier"
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
	appierRequest := *request

	// Updating app extension
	if appierRequest.App != nil {

		// *appierRequest.App creates a copy of the object in appCopy -> correct way.
		// if we do appierRequest.App just copies the reference -> Not the correct way because
		// if any of the nested property is changed it change others references to and leads to
		// change in other DSPs bidder requests as well.
		appCopy := *appierRequest.App
		appCopy.Ext, err = json.Marshal(appierAppExt{
			AppStoreID: appierRequest.App.Bundle,
		})
		if err != nil {
			errs = append(errs, err)
		}
		appierRequest.App = &appCopy
	}

	requestImpCopy := appierRequest.Imp

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

		var appierExt openrtb_ext.ExtImpTJXAppier
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

		appierRequest.Imp = []openrtb.Imp{thisImp}
		appierRequest.Cur = nil
		appierRequest.Ext = nil

		reqJSON, err := json.Marshal(appierRequest)
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
func (a *adapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return tjx_base.MakeBids(internalRequest, externalRequest, response)
}
