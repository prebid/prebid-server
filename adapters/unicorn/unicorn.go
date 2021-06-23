package unicorn

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Region ...
type Region string

const (
	JP Region = "jp"
)

// SKAN IDs must be lower case
var unicornExtSKADNetIDs = map[string]bool{
	"578prtvx9j.skadnetwork": true,
}

type adapter struct {
	endpoint         string
	SupportedRegions map[Region]string
}

// unicornImpExt is imp ext for UNICORN
type unicornImpExt struct {
	Context *unicornImpExtContext     `json:"context,omitempty"`
	Bidder  openrtb_ext.ExtImpUnicorn `json:"bidder"`
	SKADN   *openrtb_ext.SKADN        `json:"skadn,omitempty"`
}

type unicornImpExtContext struct {
	Data interface{} `json:"data,omitempty"`
}

type unicornBannerExt struct {
	Rewarded                int  `json:"rewarded"`
	AllowsCustomCloseButton bool `json:"allowscustomclosebutton"`
}

// unicornExt is ext for UNICORN
type unicornExt struct {
	Prebid    *openrtb_ext.ExtImpPrebid `json:"prebid,omitempty"`
	AccountID int64                     `json:"accountId,omitempty"`
}

type unicornVideoExt struct {
	Rewarded int `json:"rewarded"`
}

// Builder builds a new instance of the UNICORN adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
		SupportedRegions: map[Region]string{
			JP: config.XAPI.EndpointJP,
		},
	}
	return bidder, nil
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var extRegs openrtb_ext.ExtRegs
	if request.Regs != nil {
		if request.Regs.COPPA == 1 {
			return nil, []error{&errortypes.BadInput{
				Message: "COPPA is not supported",
			}}
		}
		if err := json.Unmarshal(request.Regs.Ext, &extRegs); err == nil {
			if extRegs.GDPR != nil && (*extRegs.GDPR == 1) {
				return nil, []error{&errortypes.BadInput{
					Message: "GDPR is not supported",
				}}
			}
			if extRegs.USPrivacy != "" {
				return nil, []error{&errortypes.BadInput{
					Message: "CCPA is not supported",
				}}
			}
		}
	}

	numRequests := len(request.Imp)

	requestData := make([]*adapters.RequestData, 0, numRequests)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	errs := make([]error, 0, numRequests)

	// clone the request imp array
	requestImpCopy := request.Imp

	var err error

	for i := 0; i < numRequests; i++ {
		skanSent := false

		// clone current imp
		thisImp := requestImpCopy[i]

		// extract bidder extension
		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(thisImp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("Error while decoding imp[%d].ext: %s", i, err),
			})
			continue
		}

		// unmarshal bidder extension to unicorn extension
		var unicornExt openrtb_ext.ExtImpUnicorn
		if err = json.Unmarshal(bidderExt.Bidder, &unicornExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		if thisImp.Banner != nil {
			if unicornExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner

				bannerExt := unicornBannerExt{
					Rewarded:                unicornExt.Reward,
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

		if thisImp.Video != nil {
			videoCopy := *thisImp.Video

			videoExt := unicornVideoExt{
				Rewarded: unicornExt.Reward,
			}

			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			thisImp.Video = &videoCopy
		}

		impExt := unicornImpExt{}

		if unicornExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, unicornExtSKADNetIDs)
			// only add if present
			if len(skadn.SKADNetIDs) > 0 {
				impExt.SKADN = &skadn
				skanSent = true
			}
		}

		impExt.Bidder = openrtb_ext.ExtImpUnicorn{
			PlacementID: unicornExt.PlacementID,
			PublisherID: unicornExt.PublisherID,
			MediaID:     unicornExt.MediaID,
			AccountID:   unicornExt.AccountID,
		}

		thisImp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// reinit the values in the request object
		request.Imp = []openrtb2.Imp{thisImp}

		var modifiableSource openrtb2.Source
		if request.Source != nil {
			modifiableSource = *request.Source
		} else {
			modifiableSource = openrtb2.Source{}
		}
		modifiableSource.Ext = setSourceExt()
		request.Source = &modifiableSource

		request.Ext, err = setExt(request)
		if err != nil {
			return nil, []error{err}
		}

		// json marshal the request
		reqJSON, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// assign the default uri
		uri := a.endpoint

		// assign a region based uri if it exists
		if endpoint, ok := a.SupportedRegions[Region(unicornExt.Region)]; ok {
			uri = endpoint
		}

		// Tapjoy Record placement type
		placementType := adapters.Interstitial
		if unicornExt.Reward == 1 {
			placementType = adapters.Rewarded
		}

		// build request data object
		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        "unicorn",
				PlacementType: placementType,
				Region:        unicornExt.Region,
				SKAN: adapters.SKAN{
					Supported: unicornExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: unicornExt.MRAIDSupported,
				},
			},
		}

		// append to request data array
		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

func getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")

	if request.Device != nil {
		if len(request.Device.UA) > 0 {
			headers.Add("User-Agent", request.Device.UA)
		}

		if len(request.Device.IPv6) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}

		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}
	}

	return headers
}

func modifyImps(request *openrtb2.BidRequest) error {
	for i := 0; i < len(request.Imp); i++ {
		imp := &request.Imp[i]

		var ext unicornImpExt
		err := json.Unmarshal(imp.Ext, &ext)

		if err != nil {
			return &errortypes.BadInput{
				Message: fmt.Sprintf("Error while decoding imp[%d].ext: %s", i, err),
			}
		}

		if ext.Bidder.PlacementID == "" {
			ext.Bidder.PlacementID, err = getStoredRequestImpID(imp)
			if err != nil {
				return &errortypes.BadInput{
					Message: fmt.Sprintf("Error get StoredRequestImpID from imp[%d]: %s", i, err),
				}
			}
		}

		imp.Ext, err = json.Marshal(ext)
		if err != nil {
			return &errortypes.BadInput{
				Message: fmt.Sprintf("Error while encoding imp[%d].ext: %s", i, err),
			}
		}

		secure := int8(1)
		imp.Secure = &secure
		imp.TagID = ext.Bidder.PlacementID
	}
	return nil
}

func getStoredRequestImpID(imp *openrtb2.Imp) (string, error) {
	v, err := jsonparser.GetString(imp.Ext, "prebid", "storedrequest", "id")

	if err != nil {
		return "", fmt.Errorf("stored request id not found: %s", err)
	}

	return v, nil
}

func setSourceExt() json.RawMessage {
	return json.RawMessage(`{"stype": "prebid_server_uncn", "bidder": "unicorn"}`)
}

func setExt(request *openrtb2.BidRequest) (json.RawMessage, error) {
	accountID, err := jsonparser.GetInt(request.Imp[0].Ext, "bidder", "accountId")
	if err != nil {
		accountID = 0
	}
	var decodedExt *unicornExt
	err = json.Unmarshal(request.Ext, &decodedExt)
	if err != nil {
		decodedExt = &unicornExt{
			Prebid: nil,
		}
	}
	decodedExt.AccountID = accountID

	ext, err := json.Marshal(decodedExt)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error while encoding ext, err: %s", err),
		}
	}
	return ext, nil
}

// MakeBids unpacks the server's response into Bids.
func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected http status code: 400",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected http status code: %d", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for _, bid := range seatBid.Bid {
			bid := bid
			var bidType openrtb_ext.BidType
			for _, imp := range request.Imp {
				if imp.ID == bid.ImpID {
					if imp.Banner != nil {
						bidType = openrtb_ext.BidTypeBanner
					}
				}
			}
			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}
