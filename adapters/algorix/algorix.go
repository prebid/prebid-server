package algorix

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"net/http"
	"text/template"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AlgoriXAdapter struct {
	EndpointTemplate template.Template
}

// Builder builds a new instance of the Foo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	bidder := &AlgoriXAdapter{
		EndpointTemplate: *template,
	}
	return bidder, nil
}

// MakeRequests Make Requests
func (adapter *AlgoriXAdapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapterRequests []*adapters.RequestData
	var errs []error

	adapterRequest, err := adapter.makeRequest(request)
	if err == nil {
		adapterRequests = append(adapterRequests, adapterRequest)
	} else {
		errs = append(errs, err)
	}
	return adapterRequests, errs
}

func (adapter *AlgoriXAdapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, error) {
	if len(request.Imp) == 0 {
		return nil, &errortypes.BadInput{Message: "No impression in the request"}
	}

	algorixExt := parseAlgoriXExt(request)

	if algorixExt == nil {
		return nil, &errortypes.BadInput{Message: "Invalid ExtImpAlgoriX value"}
	}

	endPoint, err := adapter.getEndPoint(algorixExt)
	if err != nil {
		return nil, err
	}

	preProcess(request)
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     endPoint,
		Body:    reqBody,
		Headers: headers,
	}, nil
}

//parseAlgoriXExt parse AlgoriX Ext
func parseAlgoriXExt(request *openrtb2.BidRequest) *openrtb_ext.ExtImpAlgorix {
	var extImpAlgoriX openrtb_ext.ExtImpAlgorix
	for _, imp := range request.Imp {
		var extBidder adapters.ExtImpBidder
		err := json.Unmarshal(imp.Ext, &extBidder)
		if err != nil {
			continue
		}
		err = json.Unmarshal(extBidder.Bidder, &extImpAlgoriX)
		if err != nil || len(extImpAlgoriX.Sid) == 0 || len(extImpAlgoriX.Token) == 0 {
			continue
		}
		return &extImpAlgoriX
	}
	return nil
}

// getEndPoint get Endpoint
func (adapter *AlgoriXAdapter) getEndPoint(ext *openrtb_ext.ExtImpAlgorix) (string, error) {
	endPointParams := macros.EndpointTemplateParams{SourceId: ext.Sid, AccountID: ext.Token}
	return macros.ResolveMacros(adapter.EndpointTemplate, endPointParams)
}

func preProcess(request *openrtb2.BidRequest) {
	for i := range request.Imp {
		if request.Imp[i].Banner != nil {
			banner := *request.Imp[i].Banner
			if (banner.W == nil || banner.H == nil || *banner.W == 0 || *banner.H == 0) && len(banner.Format) > 0 {
				firstFormat := banner.Format[0]
				bannerCopy := *request.Imp[i].Banner
				bannerCopy.W = &firstFormat.W
				bannerCopy.H = &firstFormat.H
				request.Imp[i].Banner = &bannerCopy
			}
		}
	}
}

func (adapter *AlgoriXAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, seatBid := range bidResp.SeatBid {
		for idx := range seatBid.Bid {
			mediaType := getBidType(seatBid.Bid[idx].ImpID, internalRequest.Imp)
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[idx],
				BidType: mediaType,
			})
		}
	}
	return bidResponse, nil
}

// getBidType get Bid Type
func getBidType(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Native != nil {
				return openrtb_ext.BidTypeNative
			}
			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			}
		}
	}
	return openrtb_ext.BidTypeBanner
}
