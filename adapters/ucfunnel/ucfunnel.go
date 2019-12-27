package ucfunnel

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

type UcfunnelAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

func NewUcfunnelAdapter(config *adapters.HTTPAdapterConfig, endpoint string) *UcfunnelAdapter {
	return NewUcfunnelBidder(adapters.NewHTTPAdapter(config).Client, endpoint)
}

func NewUcfunnelBidder(client *http.Client, endpoint string) *UcfunnelAdapter {
	clientAdapter := &adapters.HTTPAdapter{Client: client}
	return &UcfunnelAdapter{
		http: clientAdapter,
		URI:  endpoint,
	}
}

func (a *UcfunnelAdapter) Name() string {
	return "ucfunnel"
}

func (a *UcfunnelAdapter) SkipNoCookies() bool {
	return false
}

func (a *UcfunnelAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var errs []error
	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	var bidReq openrtb.BidRequest
	if err := json.Unmarshal(externalRequest.Body, &bidReq); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType := getBidType(bidReq, sb.Bid[i].ImpID)
			b := &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, errs
}

func (a *UcfunnelAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	// If all the requests were malformed, don't bother making a server call with no impressions.
	if len(request.Imp) == 0 {
		return nil, errs
	}

	partnerId := getPartnerId(request)
	if len(partnerId) == 0 {
		return nil, []error{}
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")

	uri := a.URI + partnerId + "/request"
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     uri,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

func getPartnerId(request *openrtb.BidRequest) string {
	var ext ExtBidderUcfunnel
	err := json.Unmarshal(request.Imp[0].Ext, &ext)
	if err != nil {
		return ""
	}
	return ext.Bidder.PartnerId
}

func AddHeadersToRequest() http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return headers
}

func getBidType(bidReq openrtb.BidRequest, impid string) openrtb_ext.BidType {
	for i := range bidReq.Imp {
		if bidReq.Imp[i].ID == impid {
			if bidReq.Imp[i].Banner != nil {
				return openrtb_ext.BidTypeBanner
			} else if bidReq.Imp[i].Video != nil {
				return openrtb_ext.BidTypeVideo
			}
		}
	}
	return openrtb_ext.BidTypeBanner
}

type ExtBidderUcfunnel struct {
	Bidder openrtb_ext.ExtImpUcfunnel `json:"bidder"`
}
