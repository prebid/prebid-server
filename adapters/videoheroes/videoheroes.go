package videoheroes

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

// Builder builds a new instance of the VideoHeroes adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	uri, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint: uri,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var videoHeroesExt *openrtb_ext.ExtImpVideoHeroes
	var err error

	videoHeroesExt, err = a.getImpressionExt(&request.Imp[0])
	if err != nil {
		return nil, []error{err}
	}

	request.Imp[0].Ext = nil

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	url, err := a.buildEndpointURL(videoHeroesExt)
	if err != nil {
		return nil, []error{err}
	}

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Body:    reqJSON,
		Uri:     url,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, nil
}

func (a *adapter) getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpVideoHeroes, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
	}
	var videoHeroesExt openrtb_ext.ExtImpVideoHeroes
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &videoHeroesExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
	}
	return &videoHeroesExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtImpVideoHeroes) (string, error) {
	endpointParams := macros.EndpointTemplateParams{PublisherID: params.PlacementID}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *adapter) MakeBids(
	receivedRequest *openrtb2.BidRequest,
	bidderRequest *adapters.RequestData,
	bidderResponse *adapters.ResponseData,
) (
	*adapters.BidderResponse,
	[]error,
) {

	if bidderResponse.StatusCode == http.StatusNoContent {
		return nil, []error{&errortypes.BadInput{Message: "No bid"}}
	}

	if bidderResponse.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", bidderResponse.StatusCode),
		}}
	}

	if bidderResponse.StatusCode == http.StatusServiceUnavailable {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Service Unavailable. Status Code: [ %d ] ", bidderResponse.StatusCode),
		}}
	}

	if bidderResponse.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Something went wrong, please contact your Account Manager. Status Code: [ %d ] ", bidderResponse.StatusCode),
		}}
	}

	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(bidderResponse.Body, &bidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	if len(bidResponse.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Empty SeatBid array",
		}}
	}

	bidResponseFinal := adapters.NewBidderResponseWithBidsCapacity(len(bidResponse.SeatBid[0].Bid))
	sb := bidResponse.SeatBid[0]

	for _, bid := range sb.Bid {
		bidResponseFinal.Bids = append(bidResponseFinal.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: getMediaTypeForImp(bid.ImpID, receivedRequest.Imp),
		})
	}
	return bidResponseFinal, nil
}

func getMediaTypeForImp(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType
		}
	}
	return mediaType
}
