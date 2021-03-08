package unicorn

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
}

// unicornImpExt is imp ext for UNICORN
type unicornImpExt struct {
	Context *unicornImpExtContext     `json:"context,omitempty"`
	Bidder  openrtb_ext.ExtImpUnicorn `json:"bidder"`
}

type unicornImpExtContext struct {
	Data interface{} `json:"data,omitempty"`
}

// unicornExt is ext for UNICORN
type unicornExt struct {
	Prebid    *openrtb_ext.ExtImpPrebid `json:"prebid,omitempty"`
	AccountID int64                     `json:"accountId,omitempty"`
}

// Builder builds a new instance of the UNICORN adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *adapter) MakeRequests(request *openrtb.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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

	err := modifyImps(request)
	if err != nil {
		return nil, []error{err}
	}

	var modifiableSource openrtb.Source
	if request.Source != nil {
		modifiableSource = *request.Source
	} else {
		modifiableSource = openrtb.Source{}
	}
	modifiableSource.Ext = setSourceExt()
	request.Source = &modifiableSource

	request.Ext, err = setExt(request)
	if err != nil {
		return nil, []error{err}
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: getHeaders(request),
	}

	return []*adapters.RequestData{requestData}, nil
}

func getHeaders(request *openrtb.BidRequest) http.Header {
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

func modifyImps(request *openrtb.BidRequest) error {
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

func getStoredRequestImpID(imp *openrtb.Imp) (string, error) {
	v, err := jsonparser.GetString(imp.Ext, "prebid", "storedrequest", "id")

	if err != nil {
		return "", fmt.Errorf("stored request id not found: %s", err)
	}

	return v, nil
}

func setSourceExt() json.RawMessage {
	return json.RawMessage(`{"stype": "prebid_server_uncn", "bidder": "unicorn"}`)
}

func setExt(request *openrtb.BidRequest) (json.RawMessage, error) {
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
func (a *adapter) MakeBids(request *openrtb.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {

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

	var response openrtb.BidResponse
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
