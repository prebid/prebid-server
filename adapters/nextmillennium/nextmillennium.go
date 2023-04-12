package nextmillennium

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
}

type nmExtPrebidStoredRequest struct {
	ID string `json:"id"`
}
type nmExtPrebid struct {
	StoredRequest nmExtPrebidStoredRequest `json:"storedrequest"`
}
type nextMillJsonExt struct {
	Prebid nmExtPrebid `json:"prebid"`
}

// MakeRequests prepares request information for prebid-server core
func (adapter *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	resImps, err := getImpressionsInfo(request.Imp)
	if len(err) > 0 {
		return nil, err
	}

	result := make([]*adapters.RequestData, 0, len(resImps))
	for _, imp := range resImps {
		bidRequest, err := adapter.buildAdapterRequest(request, imp)
		if err != nil {
			return nil, []error{err}
		}
		result = append(result, bidRequest)
	}

	return result, nil
}

func getImpressionsInfo(imps []openrtb2.Imp) (resImps []*openrtb_ext.ImpExtNextMillennium, errors []error) {
	for _, imp := range imps {
		impExt, err := getImpressionExt(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		resImps = append(resImps, impExt)
	}

	return
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ImpExtNextMillennium, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	var nextMillenniumExt openrtb_ext.ImpExtNextMillennium
	if err := json.Unmarshal(bidderExt.Bidder, &nextMillenniumExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	return &nextMillenniumExt, nil
}

func (adapter *adapter) buildAdapterRequest(prebidBidRequest *openrtb2.BidRequest, params *openrtb_ext.ImpExtNextMillennium) (*adapters.RequestData, error) {
	newBidRequest := createBidRequest(prebidBidRequest, params)

	reqJSON, err := json.Marshal(newBidRequest)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     adapter.endpoint,
		Body:    reqJSON,
		Headers: headers}, nil
}

func createBidRequest(prebidBidRequest *openrtb2.BidRequest, params *openrtb_ext.ImpExtNextMillennium) *openrtb2.BidRequest {
	placementID := params.PlacementID

	if params.GroupID != "" {
		domain := ""
		size := ""

		if prebidBidRequest.Site != nil {
			domain = prebidBidRequest.Site.Domain
		}
		if prebidBidRequest.App != nil {
			domain = prebidBidRequest.App.Domain
		}

		if banner := prebidBidRequest.Imp[0].Banner; banner != nil {
			if len(banner.Format) > 0 {
				size = fmt.Sprintf("%dx%d", banner.Format[0].W, banner.Format[0].H)
			} else if banner.W != nil && banner.H != nil {
				size = fmt.Sprintf("%dx%d", *banner.W, *banner.H)
			}
		}

		placementID = fmt.Sprintf("g%s;%s;%s", params.GroupID, size, domain)
	}
	ext := nextMillJsonExt{}
	ext.Prebid.StoredRequest.ID = placementID
	jsonExt, err := json.Marshal(ext)
	if err != nil {
		return prebidBidRequest
	}
	bidRequest := *prebidBidRequest
	bidRequest.Ext = jsonExt
	bidRequest.Imp[0].Ext = jsonExt
	return &bidRequest
}

// MakeBids translates NextMillennium bid response to prebid-server specific format
func (adapter *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var msg = ""
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		msg = fmt.Sprintf("Unexpected http status code: %d", response.StatusCode)
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}

	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		msg = fmt.Sprintf("Bad server response: %d", err)
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}
	return bidResponse, nil
}

// Builder builds a new instance of the NextMillennium adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: config.Endpoint,
	}, nil
}
