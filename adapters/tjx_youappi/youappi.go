package youappi

import (
	"encoding/json"

	openrtb "github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/tjx_base"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	EU     tjx_base.Region = "eu"
	APAC   tjx_base.Region = "apac"
	USEast tjx_base.Region = "us_east"
)

type youappiImpExt struct {
	Rewarded int                `json:"rewarded"`
	SKADN    *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type adapter struct {
	endpoint         string
	supportedRegions map[tjx_base.Region]string
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
		supportedRegions: map[tjx_base.Region]string{
			EU:     config.XAPI.EndpointEU,
			APAC:   config.XAPI.EndpointAPAC,
			USEast: config.XAPI.EndpointUSEast,
		},
	}
	return bidder, nil
}

// MakeRequests ...
func (adapter *adapter) MakeRequests(request *openrtb.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)

	requestData := make([]*adapters.RequestData, 0, numRequests)

	errs := make([]error, 0, numRequests)

	// copy the bidder request
	youappiRequest := *request

	// clone the request imp array
	requestImpCopy := youappiRequest.Imp

	var err error

	for i := 0; i < numRequests; i++ {
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

		// unmarshal bidder extension to youappi extension
		var youappiExt openrtb_ext.ExtImpTJXYouAppi
		if err = json.Unmarshal(bidderExt.Bidder, &youappiExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// remove banner if mraid is not supported
		if thisImp.Banner != nil && !youappiExt.MRAIDSupported {
			thisImp.Banner = nil
		}

		// overwrite bid floor if present
		if youappiExt.BidFloor != nil {
			thisImp.BidFloor = *youappiExt.BidFloor
		}

		impExt := youappiImpExt{
			Rewarded: youappiExt.Reward,
		}

		skanSent := false

		// add skadn if supported and present
		if youappiExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, map[string]bool{
				"3rd42ekr43.skadnetwork": true,
			})

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

		// reinit the values in the request object
		youappiRequest.Imp = []openrtb.Imp{thisImp}
		youappiRequest.Ext = nil

		// json marshal the request
		reqJSON, err := json.Marshal(youappiRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// build request data object
		reqData := &adapters.RequestData{
			Uri:     tjx_base.GetEndpointForRegion(adapter.endpoint, youappiExt.Region, adapter.supportedRegions),
			Headers: tjx_base.GetDefaultHeaders(),

			Method: "POST",
			Body:   reqJSON,

			TapjoyData: adapters.TapjoyData{
				Bidder:        "youappi",
				Region:        youappiExt.Region,
				PlacementType: tjx_base.GetPlacementType(youappiExt.Reward),

				SKAN: adapters.SKAN{
					Sent:      skanSent,
					Supported: youappiExt.SKADNSupported,
				},

				MRAID: adapters.MRAID{
					Supported: youappiExt.MRAIDSupported,
				},
			},
		}

		// append to request data array
		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

func (adapter *adapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return tjx_base.MakeBids(internalRequest, externalRequest, response)
}
