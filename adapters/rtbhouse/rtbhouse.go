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
func (adapter *RTBHouseAdapter) MakeRequests(
	openRTBRequest *openrtb2.BidRequest,
	reqInfo *adapters.ExtraRequestInfo,
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
	internalRequest *openrtb2.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData,
) (
	*adapters.BidderResponse,
	[]error,
) {
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

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))

	var bidReq openrtb2.BidRequest
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
