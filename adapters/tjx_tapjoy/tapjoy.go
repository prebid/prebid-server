package tapjoy

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/config"

	openrtb "github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

var tapjoySKADNetIDs = map[string]bool{
	"ecpz2srf59.skadnetwork": true,
}

type adapter struct {
	endpoint string
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

func (a *adapter) MakeRequests(request *openrtb.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	// copy the bidder request
	requestCopy := *request

	numRequests := len(requestCopy.Imp)

	requestData := make([]*adapters.RequestData, 0, numRequests)

	headers := http.Header{}
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")
	headers.Add("Content-Type", "application/json")

	errs := make([]error, 0, len(request.Imp))

	var err error

	requestImpCopy := requestCopy.Imp

	var srcExt *reqSourceExt
	if request.Source != nil && request.Source.Ext != nil {
		if err := json.Unmarshal(request.Source.Ext, &srcExt); err != nil {
			errs = append(errs, err)
		}
	}

	for i := 0; i < numRequests; i++ {
		thisImp := requestImpCopy[i]

		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(thisImp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		var tapjoyExt openrtb_ext.ExtImpTJXTapjoy
		if err = json.Unmarshal(bidderExt.Bidder, &tapjoyExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// This check is for identifying if the request comes from TJX
		if srcExt != nil && srcExt.HeaderBidding == 1 {
			requestCopy.BApp = nil
			requestCopy.BAdv = nil

			if tapjoyExt.Blocklist.BApp != nil {
				requestCopy.BApp = tapjoyExt.Blocklist.BApp
			}
			if tapjoyExt.Blocklist.BAdv != nil {
				requestCopy.BAdv = tapjoyExt.Blocklist.BAdv
			}
		}

		// this is important as its used by optsoa and ds teams
		// and the request.id that is passed in is generated on
		// the fly but the actual mediator request.id is needed
		// here
		requestCopy.ID = tapjoyExt.Request.ID

		// ensure correct value for request.imp[].displaymanager
		thisImp.DisplayManager = "tapjoy_sdk"

		// Overwrite BidFloor if present
		if tapjoyExt.BidFloor != nil {
			thisImp.BidFloor = *tapjoyExt.BidFloor
		}

		// request.imp[].ext
		thisImp.Ext = tapjoyExt.Extensions.ImpExt

		// request.imp[].video.ext
		if thisImp.Video != nil {
			impVideoCopy := *thisImp.Video
			impVideoCopy.Ext = tapjoyExt.Extensions.VideoExt
			thisImp.Video = &impVideoCopy
		}

		// request.app.ext
		if requestCopy.App != nil {
			appCopy := *requestCopy.App
			appCopy.Ext = tapjoyExt.Extensions.AppExt

			// overwrite app id with correct app id
			appCopy.ID = tapjoyExt.App.ID

			requestCopy.App = &appCopy
		}

		// request.device.ext + optsoa mediator device params
		if requestCopy.Device != nil {
			deviceCopy := *requestCopy.Device
			deviceCopy.Ext = tapjoyExt.Extensions.DeviceExt

			deviceCopy.OS = tapjoyExt.Device.OS
			deviceCopy.OSV = tapjoyExt.Device.OSV
			deviceCopy.HWV = tapjoyExt.Device.HWV
			deviceCopy.Make = tapjoyExt.Device.Make
			deviceCopy.Model = tapjoyExt.Device.Model
			deviceCopy.DeviceType = openrtb.DeviceType(tapjoyExt.Device.DeviceType)

			if deviceCopy.Geo != nil {
				deviceGeoCopy := *deviceCopy.Geo

				deviceGeoCopy.Metro = "0"
				deviceGeoCopy.Country = tapjoyExt.Device.CountryAlpha2

				deviceCopy.Geo = &deviceGeoCopy
			}

			requestCopy.Device = &deviceCopy
		}

		// request.app.publisher.ext
		if requestCopy.App != nil && requestCopy.App.Publisher != nil {
			publisherCopy := *requestCopy.App.Publisher
			publisherCopy.Ext = tapjoyExt.Extensions.PublisherExt
			requestCopy.App.Publisher = &publisherCopy
		}

		// request.regs.ext
		if requestCopy.Regs != nil {
			regsCopy := *requestCopy.Regs
			regsCopy.Ext = tapjoyExt.Extensions.RegsExt
			requestCopy.Regs = &regsCopy
		}

		// request.source.ext
		if requestCopy.Source != nil {
			sourceCopy := *requestCopy.Source
			sourceCopy.Ext = tapjoyExt.Extensions.SourceExt
			requestCopy.Source = &sourceCopy
		}

		// request.user.ext
		if requestCopy.User != nil {
			userCopy := *requestCopy.User
			userCopy.Ext = tapjoyExt.Extensions.UserExt
			requestCopy.User = &userCopy
		}

		// request.ext
		requestCopy.Ext = tapjoyExt.Extensions.RequestExt

		// mraid
		if thisImp.Banner != nil && !tapjoyExt.MRAIDSupported {
			thisImp.Banner = nil
		}

		// add skadn if supported and present
		skanSent := false
		if tapjoyExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, tapjoySKADNetIDs)
			if len(skadn.SKADNetIDs) > 0 {
				skanSent = true
			}
		}

		requestCopy.Imp = []openrtb.Imp{thisImp}

		reqJSON, err := json.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		placementType := adapters.Interstitial
		if tapjoyExt.Reward == 1 {
			placementType = adapters.Rewarded
		}

		reqData := &adapters.RequestData{
			Uri:     a.endpoint,
			Body:    reqJSON,
			Method:  "POST",
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        string(openrtb_ext.BidderTapjoy),
				Region:        tapjoyExt.Region,
				PlacementType: placementType,

				SKAN: adapters.SKAN{
					Supported: tapjoyExt.SKADNSupported,
					Sent:      skanSent,
				},

				MRAID: adapters.MRAID{
					Supported: tapjoyExt.MRAIDSupported,
				},
			},
		}

		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

type bidExt struct {
	Tapjoy tapjoy `json:"tapjoy,omitempty"`
}

type tapjoy struct {
	NBR            *int8           `json:"nbr,omitempty"`
	ResponseExt    json.RawMessage `json:"response_ext,omitempty"`
	ResponseBidExt json.RawMessage `json:"response_bid_ext,omitempty"`
}

// MakeBids ...
func (a *adapter) MakeBids(_ *openrtb.BidRequest, externalRequest *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	// code 204
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	// code 400
	if responseData.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", responseData.StatusCode),
		}}
	}

	// invalid code (not 200)
	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", responseData.StatusCode),
		}}
	}

	// unmarshall response body to openrtb bid response
	var tjResponse openrtb.BidResponse
	if err := json.Unmarshal(responseData.Body, &tjResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	// must have at least one bid
	if len(tjResponse.SeatBid) == 0 {
		return nil, nil
	}

	// get bid type
	bidType, err := getBidType(externalRequest)
	if err != nil {
		return nil, []error{err}
	}

	// create new prebid response object
	prebidResponse := adapters.NewBidderResponseWithBidsCapacity(len(tjResponse.SeatBid[0].Bid))

	// we should only ever have one seatbid and one bid
	for _, sb := range tjResponse.SeatBid {
		for _, b := range sb.Bid {
			if b.Price != 0 {
				// copy response.id to response.seatbid[].bid[].id
				b.ID = tjResponse.ID

				// create new bid extension with the following:
				//
				// - response.ext
				// - response.seatbid[].bid[].ext
				bidExt := bidExt{
					Tapjoy: tapjoy{
						NBR:            getNBR(tjResponse.NBR),
						ResponseExt:    tjResponse.Ext,
						ResponseBidExt: b.Ext,
					},
				}

				// overwrite the bid extension with our custom extension
				b.Ext, err = json.Marshal(&bidExt)
				if err != nil {
					return nil, []error{err}
				}

				// append to bids
				prebidResponse.Bids = append(prebidResponse.Bids, &adapters.TypedBid{
					Bid:     &b,
					BidType: bidType,
				})
			}
		}
	}

	return prebidResponse, nil
}

func getNBR(nbr *openrtb.NoBidReasonCode) *int8 {
	if nbr == nil {
		return nil
	}

	newNbr := int8(*nbr)

	return &newNbr
}

func getBidType(externalRequest *adapters.RequestData) (openrtb_ext.BidType, error) {
	var request openrtb.BidRequest
	if err := json.Unmarshal(externalRequest.Body, &request); err != nil {
		return "", err
	}

	if request.Imp[0].Video != nil {
		return openrtb_ext.BidTypeVideo, nil
	}

	return openrtb_ext.BidTypeBanner, nil
}
