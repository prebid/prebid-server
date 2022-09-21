package taurusx

import (
	"encoding/json"
	"net/http"

	"github.com/prebid/prebid-server/config"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/tjx_base"
	"github.com/prebid/prebid-server/cache/skanidlist"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Region ...
type Region string

const (
	USEast Region = "us_east"
	JP     Region = "jp"
	SG     Region = "sg"
)

type taurusxVideoExt struct {
	Rewarded int `json:"rewarded"`
}

type taurusxBannerExt struct {
	Rewarded                int  `json:"rewarded"`
	AllowsCustomCloseButton bool `json:"allowscustomclosebutton"`
}

type taurusxImpExt struct {
	SKADN *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type adapter struct {
	endpoint         string
	SupportedRegions map[Region]string
}

type reqSourceExt struct {
	HeaderBidding int `json:"header_bidding,omitempty"`
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
		SupportedRegions: map[Region]string{
			USEast: config.XAPI.EndpointUSEast,
			JP:     config.XAPI.EndpointJP,
			SG:     config.XAPI.EndpointSG,
		},
	}
	return bidder, nil
}

// MakeRequests ...
func (adapter *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)

	requestData := make([]*adapters.RequestData, 0, numRequests)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	errs := make([]error, 0, numRequests)

	// copy the bidder request
	taurusxRequest := *request

	// clone the request imp array
	requestImpCopy := taurusxRequest.Imp

	var err error

	for i := 0; i < numRequests; i++ {
		skanSent := false

		// clone current imp
		thisImp := requestImpCopy[i]

		// extract bidder extension
		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(thisImp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// unmarshal bidder extension to taurusx extension
		var taurusxExt openrtb_ext.ExtImpTJXTaurusX
		if err = json.Unmarshal(bidderExt.Bidder, &taurusxExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		taurusxRequest.BApp = nil
		taurusxRequest.BAdv = nil
		if taurusxExt.Blocklist.BApp != nil {
			taurusxRequest.BApp = taurusxExt.Blocklist.BApp
		}
		if taurusxExt.Blocklist.BAdv != nil {
			taurusxRequest.BAdv = taurusxExt.Blocklist.BAdv
		}

		impVideoExt := taurusxVideoExt{
			Rewarded: taurusxExt.Reward,
		}

		if thisImp.Video != nil {
			videoCopy := *thisImp.Video

			videoCopy.Ext, err = json.Marshal(&impVideoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			if taurusxExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner
				bannerExt := taurusxBannerExt{
					Rewarded:                taurusxExt.Reward,
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
		if taurusxExt.BidFloor != nil {
			thisImp.BidFloor = *taurusxExt.BidFloor
		}

		impExt := taurusxImpExt{}
		if taurusxExt.SKADNSupported {
			skanIDList := skanidlist.Get(openrtb_ext.BidderTaurusX)

			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, skanIDList)

			// only add if present
			if len(skadn.SKADNetIDs) > 0 {
				impExt.SKADN = &skadn
				skanSent = true
			}
		}

		thisImp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// reinit the values in the request object
		taurusxRequest.Imp = []openrtb2.Imp{thisImp}
		taurusxRequest.Ext = nil

		// json marshal the request
		reqJSON, err := json.Marshal(taurusxRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// assign the default uri
		uri := adapter.endpoint

		// assign a region based uri if it exists
		if endpoint, ok := adapter.SupportedRegions[Region(taurusxExt.Region)]; ok {
			uri = endpoint
		}

		// Tapjoy Record placement type
		placementType := adapters.Interstitial
		if taurusxExt.Reward == 1 {
			placementType = adapters.Rewarded
		}

		// build request data object
		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        string(openrtb_ext.BidderTaurusX),
				PlacementType: placementType,
				Region:        taurusxExt.Region,
				SKAN: adapters.SKAN{
					Supported: taurusxExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: taurusxExt.MRAIDSupported,
				},
				Blocklist: adapters.DynamicBlocklist{
					BApp: taurusxRequest.BApp,
					BAdv: taurusxRequest.BAdv,
				},
			},
		}

		// append to request data array
		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

// MakeBids ...
func (adapter *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return tjx_base.MakeBids(internalRequest, externalRequest, response)
}
