package dv360

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/config"
	"net/http"

	openrtb "github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
)

// Region ...
type Region string

const (
	USEast Region = "us_east"
)

type dv360ImpExt struct {
	Serverside int `json:"serverside"`
}

type dv360VideoExt struct {
	Rewarded int `json:"rewarded"`
}

type dv360DeviceExt struct {
	TruncatedIp int    `json:"truncated_ip"`
	IPLess      int    `json:"ip_less"`
	IFAType     string `json:"ifa_type"`
}

type adapter struct {
	http             *adapters.HTTPAdapter
	endpoint         string
	SupportedRegions map[Region]string
}

func (adapter *adapter) Name() string {
	return "dv360"
}

func (adapter *adapter) SkipNoCookies() bool {
	return false
}

func (adapter *adapter) Call(_ context.Context, _ *pbs.PBSRequest, _ *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	return pbs.PBSBidSlice{}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
		SupportedRegions: map[Region]string{
			USEast: config.Endpoint,
		},
	}
	return bidder, nil
}

func NewDV360LegacyAdapter(config *adapters.HTTPAdapterConfig, uri string) *adapter {
	return NewDV360Bidder(adapters.NewHTTPAdapter(config).Client, uri)
}

func NewDV360Bidder(client *http.Client, uri string) *adapter {
	return &adapter{
		http:     &adapters.HTTPAdapter{Client: client},
		endpoint: uri,
		SupportedRegions: map[Region]string{
			USEast: uri,
		},
	}
}

func (adapter *adapter) MakeRequests(request *openrtb.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	// number of requests
	numRequests := len(request.Imp)

	requestData := make([]*adapters.RequestData, 0, numRequests)

	// headers
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	// errors
	errs := make([]error, 0, numRequests)

	// clone the request imp array
	requestImpCopy := request.Imp

	// clone the request device
	var requestDeviceCopy openrtb.Device

	if request.Device != nil {
		requestDeviceCopy = *request.Device
	} else {
		requestDeviceCopy = openrtb.Device{}
	}

	var err error

	for i := 0; i < numRequests; i++ {
		// clone current imp
		impCopy := requestImpCopy[i]

		// extract bidder extension
		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(impCopy.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// unmarshal bidder extension to dv360 extension
		var dv360Ext openrtb_ext.ExtImpDV360
		if err = json.Unmarshal(bidderExt.Bidder, &dv360Ext); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		isTruncated := 0

		if requestDeviceCopy.IP != "" && requestDeviceCopy.IP != dv360Ext.RawIP {
			// If the value being sent through is an IPv4 and they don't match it must be truncated
			isTruncated = 1
		} else if requestDeviceCopy.IPv6 != "" && requestDeviceCopy.IPv6 != dv360Ext.RawIP {
			// If the value being sent through is an IPv6 and they don't match it must be truncated
			isTruncated = 1
		}

		additionalDeviceExt := dv360DeviceExt{
			TruncatedIp: isTruncated,
			IPLess:      0,
			IFAType:     "dpid", // Hardcoded as Generic device platform ID, ref info https://developers.google.com/display-video/ortb-spec#supported-extension-for-device-object
		}

		curDeviceExt := map[string]interface{}{}
		if requestDeviceCopy.Ext != nil {
			err = json.Unmarshal(requestDeviceCopy.Ext, &curDeviceExt)
			if err != nil {
				errs = append(errs, &errortypes.BadInput{
					Message: err.Error(),
				})
				continue
			}
		}

		// Assign new values without removing old ones
		curDeviceExt["truncated_ip"] = additionalDeviceExt.TruncatedIp
		curDeviceExt["ip_less"] = additionalDeviceExt.IPLess
		curDeviceExt["ifa_type"] = additionalDeviceExt.IFAType

		requestDeviceCopy.Ext, err = json.Marshal(curDeviceExt)
		if err != nil {
			errs = append(errs, err)
		}

		request.Device = &requestDeviceCopy

		rewarded := 0
		if dv360Ext.Reward == 1 {
			rewarded = 1
		}

		// if there is a banner object
		if impCopy.Banner != nil {
			// check if mraid is supported for this dsp
			if !dv360Ext.MRAIDSupported {
				// we don't support mraid, remove the banner object
				impCopy.Banner = nil
			}
		}

		// if we have a video object
		if impCopy.Video != nil {
			// make a copy of the video object
			videoCopy := *impCopy.Video

			// instantiate dv360 video extension
			videoExt := dv360VideoExt{
				Rewarded: rewarded,
			}

			// convert dv360 video extension to json
			// and append to copied video object
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			// assign copied video object to copied impression object
			impCopy.Video = &videoCopy
		}

		// create impression extension object
		// Hardcode serverside to 1 as per external disc https://tapjoy.atlassian.net/browse/NGS-44
		impExt := dv360ImpExt{
			Serverside: 1,
		}

		// json marshal the impression extension and apply to
		// copied impression object
		impCopy.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// apply the copied impression object as an array
		// to the request object
		request.Imp = []openrtb.Imp{impCopy}

		var requestAppCopy openrtb.App
		var requestAppPublisherCopy openrtb.Publisher

		if request.App != nil {
			requestAppCopy = *request.App
		} else {
			requestAppCopy = openrtb.App{}
		}

		if requestAppCopy.Publisher != nil {
			requestAppPublisherCopy = *requestAppCopy.Publisher
		} else {
			requestAppPublisherCopy = openrtb.Publisher{}
		}

		requestAppPublisherCopy.ID = "1011b04a93164a6db3a0158461c82433"

		requestAppCopy.Publisher = &requestAppPublisherCopy

		request.App = &requestAppCopy

		// json marshal the request
		body, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// assign the default uri
		uri := adapter.endpoint

		// assign adapter region based uri if it exists
		if endpoint, ok := adapter.SupportedRegions[Region(dv360Ext.Region)]; ok {
			uri = endpoint
		}

		// build request data object
		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    body,
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder: adapter.Name(),
				Region: dv360Ext.Region,
				SKAN: adapters.SKAN{
					Supported: dv360Ext.SKADNSupported,
					Sent:      false,
				},
				MRAID: adapters.MRAID{
					Supported: dv360Ext.MRAIDSupported,
				},
			},
		}

		// append to request data array
		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

func (adapter *adapter) MakeBids(_ *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &b,
					BidType: bidType,
				})
			}
		}
	}

	return bidResponse, nil
}
