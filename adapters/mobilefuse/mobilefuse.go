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
	"text/template"
)

type MobilefuseAdapter struct {
	EndpointTemplate template.Template
}

func NewMobilefuseBidder(endpointTemplate string) adapters.Bidder {
	parsedTemplate, errs := template.New("endpointTemplate").Parse(endpointTemplate)

	if errs != nil {
		glog.Fatal("Unable parse endpoint template: " + errs.Error())
		return nil
	}

	return &MobilefuseAdapter{EndpointTemplate: *parsedTemplate}
}

func (adapter *MobilefuseAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapterRequests []*adapters.RequestData

	adapterRequest, errs := adapter.makeRequest(request)

	if errs == nil {
		adapterRequests = append(adapterRequests, adapterRequest)
	}

	return adapterRequests, errs
}

func (adapter *MobilefuseAdapter) MakeBids(incomingRequest *openrtb.BidRequest, outgoingRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var incomingBidResponse openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &incomingBidResponse); err != nil {
		return nil, []error{err}
	}

	outgoingBidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, seatbid := range incomingBidResponse.SeatBid {
		for i := range seatbid.Bid {
			outgoingBidResponse.Bids = append(outgoingBidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatbid.Bid[i],
				BidType: adapter.getBidType(seatbid.Bid[i].ImpID, incomingRequest.Imp),
			})
		}
	}

	return outgoingBidResponse, nil
}

func (adapter *MobilefuseAdapter) makeRequest(bidRequest *openrtb.BidRequest) (*adapters.RequestData, []error) {
	var errs []error

	mobilefuseExtension, errs := adapter.getMobilefuseExtension(bidRequest)

	if errs != nil {
		return nil, errs
	}

	endpoint, err := adapter.getEndpoint(mobilefuseExtension)

	if err != nil {
		return nil, append(errs, err)
	}

	validImps := adapter.getValidImps(bidRequest, mobilefuseExtension)

	if len(validImps) == 0 {
		err := fmt.Errorf("No valid imps")
		errs = append(errs, err)
		return nil, errs
	}

	mobilefuseBidRequest := *bidRequest
	mobilefuseBidRequest.Imp = validImps
	body, err := json.Marshal(mobilefuseBidRequest)

	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     endpoint,
		Body:    body,
		Headers: headers,
	}, errs
}

func (adapter *MobilefuseAdapter) getMobilefuseExtension(request *openrtb.BidRequest) (*openrtb_ext.ExtImpMobilefuse, []error) {
	var mobilefuseImpExtension openrtb_ext.ExtImpMobilefuse
	var errs []error

	for _, imp := range request.Imp {
		var bidder_imp_extension adapters.ExtImpBidder

		err := json.Unmarshal(imp.Ext, &bidder_imp_extension)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		err = json.Unmarshal(bidder_imp_extension.Bidder, &mobilefuseImpExtension)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		break
	}

	return &mobilefuseImpExtension, errs
}

func (adapter *MobilefuseAdapter) getEndpoint(ext *openrtb_ext.ExtImpMobilefuse) (string, error) {
	publisher_id := strconv.Itoa(ext.PublisherId)

	url, errs := macros.ResolveMacros(adapter.EndpointTemplate, macros.EndpointTemplateParams{PublisherID: publisher_id})

	if errs != nil {
		return "", errs
	}

	if ext.TagidSrc == "ext" {
		url += "&tagid_src=ext"
	}

	return url, nil
}

func (adapter *MobilefuseAdapter) getValidImps(bidRequest *openrtb.BidRequest, ext *openrtb_ext.ExtImpMobilefuse) []openrtb.Imp {
	var validImps []openrtb.Imp

	for _, imp := range bidRequest.Imp {
		if imp.Banner != nil || imp.Video != nil {
			if imp.Banner != nil && imp.Video != nil {
				imp.Video = nil
			}

			imp.TagID = strconv.Itoa(ext.PlacementId)
			imp.Ext = nil
			validImps = append(validImps, imp)

			break
		}
	}

	return validImps
}

func (adapter *MobilefuseAdapter) getBidType(imp_id string, imps []openrtb.Imp) openrtb_ext.BidType {
	if imps[0].Video != nil {
		return openrtb_ext.BidTypeVideo
	}

	return openrtb_ext.BidTypeBanner
}
