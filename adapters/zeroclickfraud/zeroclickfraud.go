package zeroclickfraud

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

type ZeroClickFraudAdapter struct {
	EndpointTemplate template.Template
}

func (a *ZeroClickFraudAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	errs := make([]error, 0, len(request.Imp))
	headers := http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"application/json"},
	}

	// Pull the host and source ID info from the bidder params.
	reqImps, err := splitImpressions(request.Imp)

	if err != nil {
		errs = append(errs, err)
	}

	requests := []*adapters.RequestData{}

	for reqExt, reqImp := range reqImps {
		request.Imp = reqImp
		reqJson, err := json.Marshal(request)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		urlParams := macros.EndpointTemplateParams{Host: reqExt.Host, SourceId: strconv.Itoa(reqExt.SourceId)}
		url, err := macros.ResolveMacros(a.EndpointTemplate, urlParams)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		request := adapters.RequestData{
			Method:  "POST",
			Uri:     url,
			Body:    reqJson,
			Headers: headers}

		requests = append(requests, &request)
	}

	return requests, errs
}

/*
internal original request in OpenRTB, external = result of us having converted it (what comes out of MakeRequests)
*/
func (a *ZeroClickFraudAdapter) MakeBids(
	internalRequest *openrtb.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("ERR, bad input %d", response.StatusCode),
		}}
	} else if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("ERR, response with status %d", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponse()
	bidResponse.Currency = bidResp.Cur

	for _, seatBid := range bidResp.SeatBid {
		for i := 0; i < len(seatBid.Bid); i++ {
			bid := seatBid.Bid[i]
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: getMediaType(bid.ImpID, internalRequest.Imp),
			})
		}
	}

	return bidResponse, nil
}

func splitImpressions(imps []openrtb.Imp) (map[openrtb_ext.ExtImpZeroClickFraud][]openrtb.Imp, error) {

	var m = make(map[openrtb_ext.ExtImpZeroClickFraud][]openrtb.Imp)

	for _, imp := range imps {
		bidderParams, err := getBidderParams(&imp)
		if err != nil {
			return nil, err
		}

		m[*bidderParams] = append(m[*bidderParams], imp)
	}

	return m, nil
}

func getBidderParams(imp *openrtb.Imp) (*openrtb_ext.ExtImpZeroClickFraud, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Missing bidder ext: %s", err.Error()),
		}
	}
	var zeroclickfraudExt openrtb_ext.ExtImpZeroClickFraud
	if err := json.Unmarshal(bidderExt.Bidder, &zeroclickfraudExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Cannot Resolve host or sourceId: %s", err.Error()),
		}
	}

	if zeroclickfraudExt.SourceId < 1 {
		return nil, &errortypes.BadInput{
			Message: "Invalid/Missing SourceId",
		}
	}

	if len(zeroclickfraudExt.Host) < 1 {
		return nil, &errortypes.BadInput{
			Message: "Invalid/Missing Host",
		}
	}

	return &zeroclickfraudExt, nil
}

func getMediaType(impID string, imps []openrtb.Imp) openrtb_ext.BidType {

	bidType := openrtb_ext.BidTypeBanner

	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Video != nil {
				bidType = openrtb_ext.BidTypeVideo
				break
			} else if imp.Native != nil {
				bidType = openrtb_ext.BidTypeNative
				break
			} else {
				bidType = openrtb_ext.BidTypeBanner
				break
			}
		}
	}

	return bidType
}

func NewZeroClickFraudBidder(endpoint string) *ZeroClickFraudAdapter {
	template, err := template.New("endpointTemplate").Parse(endpoint)
	if err != nil {
		glog.Fatal("Unable to parse endpoint url template")
		return nil
	}

	return &ZeroClickFraudAdapter{EndpointTemplate: *template}
}
