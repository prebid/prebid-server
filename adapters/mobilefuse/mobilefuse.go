package mobilefuse

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"strconv"
	"strings" // TODO: remove
	"text/template"
)

// class
type MobilefuseAdapter struct {
	EndpointTemplate template.Template
}

// constructor
func NewMobilefuseBidder(endpoint_template string) adapters.Bidder {
	parsed_template, errors := template.New("endpoint_template").Parse(endpoint_template)

	if errors != nil {
		glog.Fatal("Unable parse endpoint template: " + errors.Error())
		return nil
	}

	return &MobilefuseAdapter{EndpointTemplate: *parsed_template}
}

// public method MakeRequests
func (adapter *MobilefuseAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapter_requests []*adapters.RequestData

	adapter_request, errors := adapter.makeRequest(request)

	if errors == nil {
		adapter_requests = append(adapter_requests, adapter_request)
	}

	return adapter_requests, errors
}

// public method MakeBids
func (adapter *MobilefuseAdapter) MakeBids(incoming_request *openrtb.BidRequest, outgoing_request *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d.", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", response.StatusCode),
		}}
	}

	var incoming_bid_response openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &incoming_bid_response); err != nil {
		return nil, []error{err}
	}

	outgoing_bid_response := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, seatbid := range incoming_bid_response.SeatBid {
		for i := range seatbid.Bid {
			outgoing_bid_response.Bids = append(outgoing_bid_response.Bids, &adapters.TypedBid{
				Bid:     &seatbid.Bid[i],
				BidType: adapter.getBidType(seatbid.Bid[i].ImpID, incoming_request.Imp),
			})
		}
	}

	return outgoing_bid_response, nil
}

// private method makeRequest
func (adapter *MobilefuseAdapter) makeRequest(bid_request *openrtb.BidRequest) (*adapters.RequestData, []error) {
	var errors []error

	mobilefuse_extension, errors := getMobilefuseExtension(bid_request)

	if mobilefuse_extension == nil {
		glog.Fatal("Invalid ExtImpMobilefuse value")
		return nil, errors
	}

	endpoint, error := adapter.getEndpoint(mobilefuse_extension)

	if error != nil {
		return nil, append(errors, error)
	}

	adapter.modifyBidRequest(bid_request, mobilefuse_extension)

	body, error := json.Marshal(bid_request)

	if error != nil {
		return nil, append(errors, error)
	}

	// TODO: gzip?
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     endpoint,
		Body:    body,
		Headers: headers,
	}, errors
}

// private function getMobilefuseExtension
func getMobilefuseExtension(request *openrtb.BidRequest) (*openrtb_ext.ExtImpMobilefuse, []error) {
	var mf_imp_extension openrtb_ext.ExtImpMobilefuse
	var errors []error

	for _, imp := range request.Imp {
		var bidder_imp_extension adapters.ExtImpBidder

		error := json.Unmarshal(imp.Ext, &bidder_imp_extension)

		if error != nil {
			errors = append(errors, error)
			continue
		}

		error = json.Unmarshal(bidder_imp_extension.Bidder, &mf_imp_extension)

		if error != nil {
			errors = append(errors, error)
			continue
		}

		break
	}

	return &mf_imp_extension, errors
}

// private method getEndpoint
func (adapter *MobilefuseAdapter) getEndpoint(ext *openrtb_ext.ExtImpMobilefuse) (string, error) {
	publisher_id := strconv.Itoa(ext.PublisherId)

	url, errors := macros.ResolveMacros(adapter.EndpointTemplate, macros.EndpointTemplateParams{PublisherID: publisher_id})

	if errors != nil {
		return "", errors
	}

	if ext.TagidSrc == "ext" {
		url += "&tagid_src=ext"
	}

	url = strings.Replace(url, "mfx-us-east", "danb-mfx", 1) // TODO: remove

	return url, nil
}

// private method modifyBidRequest
func (adapter *MobilefuseAdapter) modifyBidRequest(request *openrtb.BidRequest, ext *openrtb_ext.ExtImpMobilefuse) {
	placement_id := strconv.Itoa(ext.PlacementId)

	for i := range request.Imp {
		request.Imp[i].TagID = placement_id
	}
}

// private function getBidType
func (adapter *MobilefuseAdapter) getBidType(imp_id string, imps []openrtb.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID != imp_id {
			continue
		}

		if imp.Banner != nil {
			return openrtb_ext.BidTypeBanner
		} else if imp.Video != nil {
			return openrtb_ext.BidTypeVideo
		}
	}

	return openrtb_ext.BidTypeBanner
}
