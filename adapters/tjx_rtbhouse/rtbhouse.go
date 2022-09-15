package rtbhouse

import (
	"encoding/json"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/tjx_base"
	"github.com/prebid/prebid-server/config"
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

type rtbHouseVideoExt struct {
	PlacementType string `json:"placementtype"`
	Orientation   string `json:"orientation"`
	Skip          int    `json:"skip"`
	SkipDelay     int    `json:"skipdelay"`
}

type rtbHouseBannerExt struct {
	PlacementType           string `json:"placementtype"`
	AllowsCustomCloseButton bool   `json:"allowscustomclosebutton"`
}

type rtbHouseImpExt struct {
	Rewarded int                `json:"rewarded"`
	SKADN    *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type reqSourceExt struct {
	HeaderBidding int `json:"header_bidding,omitempty"`
}

// Orientation ...
type Orientation string

const (
	Horizontal Orientation = "h"
	Vertical   Orientation = "v"
)

// SKAN IDs must be lower case
var rtbHouseSKADNetIDs = map[string]bool{
	"8s468mfl3y.skadnetwork": true,
}

// Builder builds a new instance of the RTBHouse adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &RTBHouseAdapter{
		endpoint: config.Endpoint,
		SupportedRegions: map[Region]string{
			USEast: config.XAPI.EndpointUSEast,
			EU:     config.XAPI.EndpointEU,
			APAC:   config.XAPI.EndpointAPAC,
		},
	}
	return bidder, nil
}

// RTBHouseAdapter implements the Bidder interface.
type RTBHouseAdapter struct {
	endpoint         string
	SupportedRegions map[Region]string
}

func (adapter *RTBHouseAdapter) Name() string {
	return string(openrtb_ext.BidderRTBHouse)
}

// MakeRequests prepares the HTTP requests which should be made to fetch bids.
func (adapter *RTBHouseAdapter) MakeRequests(
	openRTBRequest *openrtb2.BidRequest,
	_ *adapters.ExtraRequestInfo,
) (
	[]*adapters.RequestData,
	[]error,
) {
	numRequests := len(openRTBRequest.Imp)

	requestData := make([]*adapters.RequestData, 0, numRequests)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	errs := make([]error, 0, len(openRTBRequest.Imp))
	var err error

	// copy the bidder request
	rthhouseRequest := *openRTBRequest

	requestImpCopy := rthhouseRequest.Imp

	var srcExt *reqSourceExt
	if openRTBRequest.Source != nil && openRTBRequest.Source.Ext != nil {
		if err := json.Unmarshal(openRTBRequest.Source.Ext, &srcExt); err != nil {
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

		var rtbHouseExt openrtb_ext.ExtImpTJXRTBHouse
		if err = json.Unmarshal(bidderExt.Bidder, &rtbHouseExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// This check is for identifying if the request comes from TJX
		if srcExt != nil && srcExt.HeaderBidding == 1 {
			rthhouseRequest.BApp = nil
			rthhouseRequest.BAdv = nil

			if rtbHouseExt.Blocklist.BApp != nil {
				rthhouseRequest.BApp = rtbHouseExt.Blocklist.BApp
			}
			if rtbHouseExt.Blocklist.BAdv != nil {
				rthhouseRequest.BAdv = rtbHouseExt.Blocklist.BAdv
			}
		}

		// Updating required publisher id field
		if rthhouseRequest.App != nil {
			if rthhouseRequest.App.Publisher != nil {
				rthhouseRequest.App.Publisher.ID = rtbHouseExt.PublisherID
			} else {
				publisher := openrtb2.Publisher{
					ID: rtbHouseExt.PublisherID,
				}
				rthhouseRequest.App.Publisher = &publisher
			}
		}

		// default is interstitial
		placementType := adapters.Interstitial
		rewarded := 0
		if rtbHouseExt.Video.Skip == 0 {
			placementType = adapters.Rewarded
			rewarded = 1
		}

		if thisImp.Video != nil {
			orientation := Horizontal
			if rtbHouseExt.Video.Width < rtbHouseExt.Video.Height {
				orientation = Vertical
			}

			videoCopy := *thisImp.Video

			videoExt := rtbHouseVideoExt{
				PlacementType: string(placementType),
				Orientation:   string(orientation),
				Skip:          rtbHouseExt.Video.Skip,
				SkipDelay:     rtbHouseExt.Video.SkipDelay,
			}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			if rtbHouseExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner
				bannerExt := rtbHouseBannerExt{
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
		if rtbHouseExt.BidFloor != nil {
			thisImp.BidFloor = *rtbHouseExt.BidFloor
		}

		impExt := rtbHouseImpExt{
			Rewarded: rewarded,
		}
		// Add SKADN if supported and present
		if rtbHouseExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, rtbHouseSKADNetIDs)
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

		rthhouseRequest.Imp = []openrtb2.Imp{thisImp}
		rthhouseRequest.Cur = nil
		rthhouseRequest.Ext = nil

		reqJSON, err := json.Marshal(rthhouseRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		uri := adapter.endpoint
		if endpoint, ok := adapter.SupportedRegions[Region(rtbHouseExt.Region)]; ok {
			uri = endpoint
		}

		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        adapter.Name(),
				PlacementType: placementType,
				Region:        rtbHouseExt.Region,
				SKAN: adapters.SKAN{
					Supported: rtbHouseExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: rtbHouseExt.MRAIDSupported,
				},
			},
		}

		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

// MakeBids unpacks the server's response into Bids.
func (adapter *RTBHouseAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return tjx_base.MakeBids(internalRequest, externalRequest, response)
}
