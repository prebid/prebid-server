package algorix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"text/template"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	EndpointTemplate template.Template
}

// Builder builds a new instance of the AlgoriX adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	endpoint, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	bidder := &adapter{
		EndpointTemplate: *endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapterRequests []*adapters.RequestData
	var errs []error

	adapterRequest, err := a.makeRequest(request)
	if err == nil {
		adapterRequests = append(adapterRequests, adapterRequest)
	} else {
		errs = append(errs, err)
	}
	return adapterRequests, errs
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, error) {
	algorixExt, err := getImpAlgoriXExt(&request.Imp[0])

	if err != nil {
		return nil, &errortypes.BadInput{Message: "Invalid ExtImpAlgoriX value"}
	}

	endPoint, err := a.getEndPoint(algorixExt)
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

// get ImpAlgoriXExt From First Imp. Only check and get first Imp.Ext.Bidder to ExtImpAlgorix
func getImpAlgoriXExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpAlgorix, error) {
	var extImpAlgoriX openrtb_ext.ExtImpAlgorix
	var extBidder adapters.ExtImpBidder
	err := json.Unmarshal(imp.Ext, &extBidder)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(extBidder.Bidder, &extImpAlgoriX)
	if err != nil {
		return nil, err
	}
	return &extImpAlgoriX, nil
}

func (a *adapter) getEndPoint(ext *openrtb_ext.ExtImpAlgorix) (string, error) {
	endPointParams := macros.EndpointTemplateParams{
		SourceId:  url.PathEscape(ext.Sid),
		AccountID: url.PathEscape(ext.Token),
	}
	return macros.ResolveMacros(a.EndpointTemplate, endPointParams)
}

func preProcess(request *openrtb2.BidRequest) {
	for i := range request.Imp {
		if request.Imp[i].Banner != nil {
			banner := *request.Imp[i].Banner
			if (banner.W == nil || banner.H == nil || *banner.W == 0 || *banner.H == 0) && len(banner.Format) > 0 {
				firstFormat := banner.Format[0]
				banner.W = &firstFormat.W
				banner.H = &firstFormat.H
				request.Imp[i].Banner = &banner
			}
		}
	}
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

func getBidType(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				break
			}
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
