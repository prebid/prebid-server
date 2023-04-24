package axis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint *template.Template
}

type reqBodyExt struct {
	AxisBidderExt openrtb_ext.ImpExtAxis `json:"bidder"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint: template,
	}

	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapterRequests []*adapters.RequestData

	originalImpSlice := request.Imp

	for i := range request.Imp {
		currImp := originalImpSlice[i]
		request.Imp = []openrtb2.Imp{currImp}

		var bidderExt reqBodyExt
		if err := json.Unmarshal(currImp.Ext, &bidderExt); err != nil {
			continue
		}

		url, err := a.buildEndpointURL(&bidderExt)
		if err != nil {
			return nil, []error{err}
		}

		extJson, err := json.Marshal(bidderExt)
		if err != nil {
			return nil, []error{err}
		}

		request.Imp[0].Ext = extJson

		adapterReq, err := buildRequest(request, url)
		if err != nil {
			return nil, []error{err}
		}

		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
	}
	request.Imp = originalImpSlice
	return adapterRequests, nil
}

func (a *adapter) buildEndpointURL(bidderExt *reqBodyExt) (string, error) {
	endpointParams := macros.EndpointTemplateParams{
		AccountID: bidderExt.AxisBidderExt.Integration,
		SourceId:  bidderExt.AxisBidderExt.Token,
	}

	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func buildRequest(request *openrtb2.BidRequest, url string) (*adapters.RequestData, error) {
	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     url,
		Body:    reqJSON,
		Headers: headers,
	}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	impsMappedByID := make(map[string]openrtb2.Imp, len(request.Imp))
	for i, imp := range request.Imp {
		impsMappedByID[request.Imp[i].ID] = imp
	}

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForImp(seatBid.Bid[i].ImpID, impsMappedByID)
			if err != nil {
				return nil, []error{err}
			}

			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

func getMediaTypeForImp(impID string, impMap map[string]openrtb2.Imp) (openrtb_ext.BidType, error) {
	if index, found := impMap[impID]; found {
		if index.Banner != nil {
			return openrtb_ext.BidTypeBanner, nil
		}
		if index.Video != nil {
			return openrtb_ext.BidTypeVideo, nil
		}
		if index.Native != nil {
			return openrtb_ext.BidTypeNative, nil
		}
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find impression \"%s\"", impID),
	}
}
