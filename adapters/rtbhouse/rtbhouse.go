package rtbhouse

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
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
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
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
func (adapter *RTBHouseAdapter) MakeRequests(openRTBRequest *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(openRTBRequest.Imp)

	requestData := make([]*adapters.RequestData, 0, numRequests)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	errs := make([]error, 0, len(openRTBRequest.Imp))
	var err error

	requestImpCopy := openRTBRequest.Imp

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

		var rtbHouseExt openrtb_ext.ExtImpRTBHouse
		if err = json.Unmarshal(bidderExt.Bidder, &rtbHouseExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// Updating required publisher id field
		if openRTBRequest.App != nil {
			if openRTBRequest.App.Publisher != nil {
				openRTBRequest.App.Publisher.ID = rtbHouseExt.PublisherID
			} else {
				publisher := openrtb2.Publisher{
					ID: rtbHouseExt.PublisherID,
				}
				openRTBRequest.App.Publisher = &publisher
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

		openRTBRequest.Imp = []openrtb2.Imp{thisImp}
		openRTBRequest.Cur = nil
		openRTBRequest.Ext = nil

		reqJSON, err := json.Marshal(openRTBRequest)
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

const unexpectedStatusCodeFormat = "" +
	"Unexpected status code: %d. Run with request.debug = 1 for more info"

// MakeBids unpacks the server's response into Bids.
func (adapter *RTBHouseAdapter) MakeBids(
	openRTBRequest *openrtb2.BidRequest,
	requestToBidder *adapters.RequestData,
	bidderRawResponse *adapters.ResponseData,
) (
	bidderResponse *adapters.BidderResponse,
	errs []error,
) {
	switch bidderRawResponse.StatusCode {
	case http.StatusOK:
		break
	case http.StatusNoContent:
		return nil, nil
	case http.StatusBadRequest:
		err := &errortypes.BadInput{
			Message: fmt.Sprintf(unexpectedStatusCodeFormat, bidderRawResponse.StatusCode),
		}
		return nil, []error{err}
	default:
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf(unexpectedStatusCodeFormat, bidderRawResponse.StatusCode),
		}
		return nil, []error{err}
	}

	var openRTBBidderResponse openrtb2.BidResponse
	if err := json.Unmarshal(bidderRawResponse.Body, &openRTBBidderResponse); err != nil {
		return nil, []error{err}
	}

	bidsCapacity := len(openRTBBidderResponse.SeatBid[0].Bid)
	bidderResponse = adapters.NewBidderResponseWithBidsCapacity(bidsCapacity)
	var typedBid *adapters.TypedBid
	for _, seatBid := range openRTBBidderResponse.SeatBid {
		for _, bid := range seatBid.Bid {
			bid := bid // pin! -> https://github.com/kyoh86/scopelint#whats-this
			typedBid = &adapters.TypedBid{Bid: &bid, BidType: "banner"}
			bidderResponse.Bids = append(bidderResponse.Bids, typedBid)
		}
	}

	return bidderResponse, nil

}
