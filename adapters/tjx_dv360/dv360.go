package dv360

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

type reqSourceExt struct {
	HeaderBidding int `json:"header_bidding,omitempty"`
}

type adapter struct {
	endpoint         string
	SupportedRegions map[Region]string
}

func (adapter *adapter) Name() string {
	return "dv360"
}

func (adapter *adapter) SkipNoCookies() bool {
	return false
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

	// copy the bidder request
	dv360Request := *request

	// clone the request imp array
	requestImpCopy := dv360Request.Imp

	// clone the request device
	var requestDeviceCopy openrtb.Device

	if dv360Request.Device != nil {
		requestDeviceCopy = *dv360Request.Device
	} else {
		requestDeviceCopy = openrtb.Device{}
	}

	var err error

	var srcExt *reqSourceExt
	if request.Source != nil && request.Source.Ext != nil {
		if err := json.Unmarshal(request.Source.Ext, &srcExt); err != nil {
			errs = append(errs, err)
		}
	}

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
		var dv360Ext openrtb_ext.ExtImpTJXDV360
		if err = json.Unmarshal(bidderExt.Bidder, &dv360Ext); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// This check is for identifying if the request comes from TJX
		if srcExt != nil && srcExt.HeaderBidding == 1 {
			dv360Request.BApp = nil
			dv360Request.BAdv = nil

			if dv360Ext.Blocklist.BApp != nil {
				dv360Request.BApp = dv360Ext.Blocklist.BApp
			}
			if dv360Ext.Blocklist.BAdv != nil {
				dv360Request.BAdv = dv360Ext.Blocklist.BAdv
			}
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

		dv360Request.Device = &requestDeviceCopy

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

		// Overwrite BidFloor if present
		if dv360Ext.BidFloor != nil {
			impCopy.BidFloor = *dv360Ext.BidFloor
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
		dv360Request.Imp = []openrtb.Imp{impCopy}
		dv360Request.Ext = nil

		// json marshal the request
		body, err := json.Marshal(dv360Request)
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

func (adapter *adapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return tjx_base.MakeBids(internalRequest, externalRequest, response)
}
