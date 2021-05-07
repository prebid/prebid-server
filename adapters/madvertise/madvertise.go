package madvertise

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpointTemplate template.Template
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpointTemplate: *template,
	}

	return bidder, nil
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

		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}
	}

	return headers
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	zoneID := ""
	for _, imp := range request.Imp {
		madvertiseExt, err := getImpressionExt(imp)
		if err != nil {
			return nil, []error{err}
		}
		if len(madvertiseExt.ZoneID) < 7 {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("The minLength of zone ID is 7; ImpID=%s", imp.ID),
			}}
		}
		if zoneID == "" {
			zoneID = madvertiseExt.ZoneID
		} else if zoneID != madvertiseExt.ZoneID {
			return nil, []error{&errortypes.BadInput{
				Message: "There must be only one zone ID",
			}}
		}
	}
	url, err := a.buildEndpointURL(zoneID)
	if err != nil {
		return nil, []error{err}
	}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     url,
		Body:    requestJSON,
		Headers: getHeaders(request),
	}

	return []*adapters.RequestData{requestData}, nil
}

func getImpressionExt(imp openrtb2.Imp) (*openrtb_ext.ExtImpMadvertise, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("%s; ImpID=%s", err.Error(), imp.ID),
		}
	}
	var madvertiseExt openrtb_ext.ExtImpMadvertise
	if err := json.Unmarshal(bidderExt.Bidder, &madvertiseExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("%s; ImpID=%s", err.Error(), imp.ID),
		}
	}
	if madvertiseExt.ZoneID == "" {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("ext.bidder.zoneId not provided; ImpID=%s", imp.ID),
		}
	}

	return &madvertiseExt, nil
}

func (a *adapter) buildEndpointURL(zoneID string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ZoneID: zoneID}
	return macros.ResolveMacros(a.endpointTemplate, endpointParams)
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", responseData.StatusCode),
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
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]
			bidMediaType := getMediaTypeForBid(bid.Attr)
			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidMediaType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

func getMediaTypeForBid(attr []openrtb2.CreativeAttribute) openrtb_ext.BidType {
	for i := 0; i < len(attr); i++ {
		if attr[i] == openrtb2.CreativeAttribute(16) {
			return openrtb_ext.BidTypeVideo
		} else if attr[i] == openrtb2.CreativeAttribute(6) {
			return openrtb_ext.BidTypeVideo
		} else if attr[i] == openrtb2.CreativeAttribute(7) {
			return openrtb_ext.BidTypeVideo
		}
	}
	return openrtb_ext.BidTypeBanner
}
