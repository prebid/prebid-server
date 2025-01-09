package axonix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	EndpointTemplate *template.Template
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	bidder := &adapter{
		EndpointTemplate: endpoint,
	}
	return bidder, nil
}

func (a *adapter) getEndpoint(ext *openrtb_ext.ExtImpAxonix) (string, error) {
	endpointParams := macros.EndpointTemplateParams{
		AccountID: url.PathEscape(ext.SupplyId),
	}
	return macros.ResolveMacros(a.EndpointTemplate, endpointParams)
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error

	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil {
		errors = append(errors, &errortypes.BadInput{
			Message: err.Error(),
		})

		return nil, errors
	}

	var axonixExt openrtb_ext.ExtImpAxonix
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &axonixExt); err != nil {
		errors = append(errors, &errortypes.BadInput{
			Message: err.Error(),
		})

		return nil, errors
	}

	endpoint, err := a.getEndpoint(&axonixExt)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     endpoint,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for _, bid := range seatBid.Bid {
			bid := bid
			resolveMacros(&bid)
			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: getMediaType(bid.ImpID, request.Imp),
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	return bidResponse, nil
}

func getMediaType(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Native != nil {
				return openrtb_ext.BidTypeNative
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			}
			return openrtb_ext.BidTypeBanner
		}
	}
	return openrtb_ext.BidTypeBanner
}

func resolveMacros(bid *openrtb2.Bid) {
	if bid == nil {
		return
	}
	price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
	bid.NURL = strings.Replace(bid.NURL, "${AUCTION_PRICE}", price, -1)
	bid.AdM = strings.Replace(bid.AdM, "${AUCTION_PRICE}", price, -1)
}
