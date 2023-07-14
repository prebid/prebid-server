package mobilefuse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type MobileFuseAdapter struct {
	EndpointTemplate *template.Template
}

type ExtMf struct {
	MediaType string `json:"media_type"`
}

type BidExt struct {
	Mf ExtMf `json:"mf"`
}

type ExtSkadn struct {
	Skadn json.RawMessage `json:"skadn"`
}

// Builder builds a new instance of the MobileFuse adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &MobileFuseAdapter{
		EndpointTemplate: template,
	}
	return bidder, nil
}

func (adapter *MobileFuseAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapterRequests []*adapters.RequestData

	adapterRequest, errs := adapter.makeRequest(request)

	if errs == nil {
		adapterRequests = append(adapterRequests, adapterRequest)
	}

	return adapterRequests, errs
}

func (adapter *MobileFuseAdapter) MakeBids(incomingRequest *openrtb2.BidRequest, outgoingRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var incomingBidResponse openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &incomingBidResponse); err != nil {
		return nil, []error{err}
	}

	outgoingBidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, seatbid := range incomingBidResponse.SeatBid {
		for i := range seatbid.Bid {
			bidType := getBidType(seatbid.Bid[i])
			seatbid.Bid[i].Ext = nil

			outgoingBidResponse.Bids = append(outgoingBidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatbid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return outgoingBidResponse, nil
}

func (adapter *MobileFuseAdapter) makeRequest(bidRequest *openrtb2.BidRequest) (*adapters.RequestData, []error) {
	var errs []error

	mobileFuseExtension, errs := getFirstMobileFuseExtension(bidRequest)

	if errs != nil {
		return nil, errs
	}

	endpoint, err := adapter.getEndpoint(mobileFuseExtension)

	if err != nil {
		return nil, append(errs, err)
	}

	validImps, err := getValidImps(bidRequest, mobileFuseExtension)

	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	mobileFuseBidRequest := *bidRequest
	mobileFuseBidRequest.Imp = validImps
	body, err := json.Marshal(mobileFuseBidRequest)

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

func getFirstMobileFuseExtension(request *openrtb2.BidRequest) (*openrtb_ext.ExtImpMobileFuse, []error) {
	var mobileFuseImpExtension openrtb_ext.ExtImpMobileFuse
	var errs []error

	for _, imp := range request.Imp {
		var bidder_imp_extension adapters.ExtImpBidder

		err := json.Unmarshal(imp.Ext, &bidder_imp_extension)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		err = json.Unmarshal(bidder_imp_extension.Bidder, &mobileFuseImpExtension)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		break
	}

	return &mobileFuseImpExtension, errs
}

func (adapter *MobileFuseAdapter) getEndpoint(ext *openrtb_ext.ExtImpMobileFuse) (string, error) {
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

func getValidImps(bidRequest *openrtb2.BidRequest, ext *openrtb_ext.ExtImpMobileFuse) ([]openrtb2.Imp, error) {
	var validImps []openrtb2.Imp

	for _, imp := range bidRequest.Imp {
		if imp.Banner != nil || imp.Video != nil || imp.Native != nil {
			imp.TagID = strconv.Itoa(ext.PlacementId)

			var extSkadn ExtSkadn
			err := json.Unmarshal(imp.Ext, &extSkadn)
			if err != nil {
				return nil, err
			}

			if extSkadn.Skadn != nil {
				imp.Ext, err = json.Marshal(map[string]json.RawMessage{"skadn": extSkadn.Skadn})
				if err != nil {
					return nil, err
				}
			} else {
				imp.Ext = nil
			}

			validImps = append(validImps, imp)

			break
		}
	}

	if len(validImps) == 0 {
		return nil, fmt.Errorf("No valid imps")
	}

	return validImps, nil
}

func getBidType(bid openrtb2.Bid) openrtb_ext.BidType {
	if bid.Ext != nil {
		var bidExt BidExt
		err := json.Unmarshal(bid.Ext, &bidExt)
		if err == nil {
			if bidExt.Mf.MediaType == "video" {
				return openrtb_ext.BidTypeVideo
			} else if bidExt.Mf.MediaType == "native" {
				return openrtb_ext.BidTypeNative
			}
		}
	}

	return openrtb_ext.BidTypeBanner
}
