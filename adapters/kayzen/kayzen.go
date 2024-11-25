package kayzen

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint *template.Template
}

// Builder builds a new instance of the Kayzen adapter for the given bidder with the given config.
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

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) (requests []*adapters.RequestData, errors []error) {
	var kayzenExt *openrtb_ext.ExtKayzen
	var err error

	if len(request.Imp) > 0 {
		kayzenExt, err = a.getImpressionExt(&(request.Imp[0]))
		if err != nil {
			errors = append(errors, err)
		}
		request.Imp[0].Ext = nil
	} else {
		errors = append(errors, &errortypes.BadInput{
			Message: "Missing Imp Object",
		})
	}

	if len(errors) > 0 {
		return nil, errors
	}

	url, err := a.buildEndpointURL(kayzenExt)
	if err != nil {
		return nil, []error{err}
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Body:    requestJSON,
		Uri:     url,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, nil
}

func (a *adapter) getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtKayzen, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Bidder extension not provided or can't be unmarshalled",
		}
	}
	var kayzenExt openrtb_ext.ExtKayzen
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &kayzenExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error while unmarshaling bidder extension",
		}
	}
	return &kayzenExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtKayzen) (string, error) {
	endpointParams := macros.EndpointTemplateParams{
		ZoneID:    params.Zone,
		AccountID: params.Exchange,
	}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if response.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", response.StatusCode),
		}
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	var errs []error
	for _, seatBid := range bidResp.SeatBid {
		for _, bid := range seatBid.Bid {
			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: getMediaTypeForImp(bid.ImpID, internalRequest.Imp),
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, errs
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative
			}
		}
	}
	return openrtb_ext.BidTypeBanner
}
