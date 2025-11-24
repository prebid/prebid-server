package targetVideo

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

type impExtPrebid struct {
	Prebid *openrtb_ext.ExtImpPrebid `json:"prebid,omitempty"`
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	totalImps := len(request.Imp)
	errors := make([]error, 0)
	adapterRequests := make([]*adapters.RequestData, 0, totalImps)

	// Split multi-imp request into multiple ad server requests. SRA is currently not recommended.
	for i := 0; i < totalImps; i++ {
		if adapterReq, err := a.makeRequest(*request, request.Imp[i]); err == nil {
			adapterRequests = append(adapterRequests, adapterReq)
		} else {
			errors = append(errors, err)
		}
	}

	return adapterRequests, errors
}

func (a *adapter) makeRequest(request openrtb2.BidRequest, imp openrtb2.Imp) (*adapters.RequestData, error) {

	// For now, this adapter sends one imp per request, but we still
	// iterate over all imps in the request to perform the required
	// imp.ext transformation.
	request.Imp = []openrtb2.Imp{imp}

	for i := range request.Imp {

		var extBidder adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &extBidder); err != nil {
			return nil, &errortypes.BadInput{Message: fmt.Sprintf("Invalid ext.bidder")}
		}
		var extImpTargetVideo openrtb_ext.ExtImpTargetVideo
		if err := jsonutil.Unmarshal(extBidder.Bidder, &extImpTargetVideo); err != nil {
			return nil, &errortypes.BadInput{Message: fmt.Sprintf("Placement ID missing")}
		}
		var prebid *openrtb_ext.ExtImpPrebid
		if extBidder.Prebid == nil {
			prebid = &openrtb_ext.ExtImpPrebid{}
		}
		if prebid.StoredRequest == nil {
			prebid.StoredRequest = &openrtb_ext.ExtStoredRequest{}
		}
		prebid.StoredRequest.ID = fmt.Sprintf("%d", extImpTargetVideo.PlacementId)

		ext := impExtPrebid{
			Prebid: prebid,
		}

		extRaw, err := jsonutil.Marshal(ext)
		if err != nil {
			return nil, &errortypes.BadInput{Message: fmt.Sprintf("error building imp.ext, err: %s", err)}
		}

		request.Imp[i].Ext = extRaw

	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	//fmt.Println("TARGET VIDEO reqJson: ", string(reqJSON))

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

func (a *adapter) MakeBids(bidReq *openrtb2.BidRequest, unused *adapters.RequestData, httpRes *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(httpRes) {
		return nil, nil
	}
	if statusError := adapters.CheckResponseStatusCodeForErrors(httpRes); statusError != nil {
		return nil, []error{statusError}
	}

	bidResp, errResp := prepareBidResponse(httpRes.Body)
	if errResp != nil {
		return nil, []error{errResp}
	}

	br := adapters.NewBidderResponse()
	errs := []error{}

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bid := sb.Bid[i]

			mediaType := openrtb_ext.BidTypeVideo

			br.Bids = append(br.Bids, &adapters.TypedBid{Bid: &bid, BidType: mediaType})
		}
	}
	return br, errs
}

func prepareBidResponse(body []byte) (openrtb2.BidResponse, error) {
	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(body, &response); err != nil {
		return response, err
	}
	return response, nil
}

func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {

	bidder := &adapter{
		endpoint: cfg.Endpoint,
	}
	return bidder, nil
}
