package tapjoy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/config"

	openrtb "github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
)

var tapjoySKADNetIDs = map[string]bool{
	"ecpz2srf59.skadnetwork": true,
}

type adapter struct {
	http     *adapters.HTTPAdapter
	endpoint string
}

func (a *adapter) Name() string {
	return "tapjoy"
}

func (a *adapter) SkipNoCookies() bool {
	return false
}

func (a *adapter) Call(_ context.Context, _ *pbs.PBSRequest, _ *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	return pbs.PBSBidSlice{}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}

	return bidder, nil
}

func NewTapjoyBidder(client *http.Client, uri string) *adapter {
	return &adapter{
		http:     &adapters.HTTPAdapter{Client: client},
		endpoint: uri,
	}
}

func NewTapjoyLegacyAdapter(config *adapters.HTTPAdapterConfig, uri string) *adapter {
	return NewTapjoyBidder(adapters.NewHTTPAdapter(config).Client, uri)
}

func (a *adapter) MakeRequests(request *openrtb.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requestCopy := *request

	numRequests := len(requestCopy.Imp)

	requestData := make([]*adapters.RequestData, 0, numRequests)

	headers := http.Header{}
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")
	headers.Add("Content-Type", "application/json;charset=utf-8")

	errs := make([]error, 0, len(request.Imp))

	var err error

	requestImpCopy := requestCopy.Imp

	for i := 0; i < numRequests; i++ {
		thisImp := requestImpCopy[i]

		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(thisImp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		var tapjoyExt openrtb_ext.ExtImpTapjoy
		if err = json.Unmarshal(bidderExt.Bidder, &tapjoyExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// request.imp[].ext
		thisImp.Ext, err = json.Marshal(&tapjoyExt.Extensions.ImpExt)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// request.imp[].video.ext
		if thisImp.Video != nil {
			impVideoCopy := *thisImp.Video
			impVideoCopy.Ext, err = json.Marshal(&tapjoyExt.Extensions.VideoExt)
			if err != nil {
				errs = append(errs, &errortypes.BadInput{
					Message: err.Error(),
				})
				continue
			}
			thisImp.Video = &impVideoCopy
		}

		// request.app.ext
		if requestCopy.App != nil {
			appCopy := *requestCopy.App
			appCopy.Ext, err = json.Marshal(&tapjoyExt.Extensions.AppExt)
			if err != nil {
				errs = append(errs, &errortypes.BadInput{
					Message: err.Error(),
				})
				continue
			}
			requestCopy.App = &appCopy
		}

		// request.device.ext + optsoa mediator device params
		if requestCopy.Device != nil {
			deviceCopy := *requestCopy.Device
			deviceCopy.Ext, err = json.Marshal(&tapjoyExt.Extensions.DeviceExt)
			if err != nil {
				errs = append(errs, &errortypes.BadInput{
					Message: err.Error(),
				})
				continue
			}

			deviceCopy.OS = tapjoyExt.Device.OS
			deviceCopy.OSV = tapjoyExt.Device.OSV
			deviceCopy.HWV = tapjoyExt.Device.HWV
			deviceCopy.Make = tapjoyExt.Device.Make
			deviceCopy.Model = tapjoyExt.Device.Model
			deviceCopy.DeviceType = openrtb.DeviceType(tapjoyExt.Device.DeviceType)

			requestCopy.Device = &deviceCopy
		}

		// request.app.publisher.ext
		if requestCopy.App != nil && requestCopy.App.Publisher != nil {
			publisherCopy := *requestCopy.App.Publisher
			publisherCopy.Ext, err = json.Marshal(&tapjoyExt.Extensions.PublisherExt)
			if err != nil {
				errs = append(errs, &errortypes.BadInput{
					Message: err.Error(),
				})
				continue
			}
			requestCopy.App.Publisher = &publisherCopy
		}

		// request.regs.ext
		if requestCopy.Regs != nil {
			regsCopy := *requestCopy.Regs
			regsCopy.Ext, err = json.Marshal(&tapjoyExt.Extensions.RegsExt)
			if err != nil {
				errs = append(errs, &errortypes.BadInput{
					Message: err.Error(),
				})
				continue
			}
			requestCopy.Regs = &regsCopy
		}

		// request.source.ext
		if requestCopy.Source != nil {
			sourceCopy := *requestCopy.Source
			sourceCopy.Ext, err = json.Marshal(&tapjoyExt.Extensions.SourceExt)
			if err != nil {
				errs = append(errs, &errortypes.BadInput{
					Message: err.Error(),
				})
				continue
			}
			requestCopy.Source = &sourceCopy
		}

		// request.user.ext
		if requestCopy.User != nil {
			userCopy := *requestCopy.User
			userCopy.Ext, err = json.Marshal(&tapjoyExt.Extensions.UserExt)
			if err != nil {
				errs = append(errs, &errortypes.BadInput{
					Message: err.Error(),
				})
				continue
			}
			requestCopy.User = &userCopy
		}

		// request.ext
		requestCopy.Ext, err = json.Marshal(&tapjoyExt.Extensions.RequestExt)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

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
		if tapjoyExt.Extensions.VideoExt.Rewarded == 1 {
			placementType = adapters.Rewarded
		}

		reqData := &adapters.RequestData{
			Uri:     a.endpoint,
			Body:    reqJSON,
			Method:  "POST",
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        a.Name(),
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

// MakeBids ...
func (a *adapter) MakeBids(_ *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))

	var bidReq openrtb.BidRequest
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
