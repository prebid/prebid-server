package ucfunnel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type UcfunnelAdapter struct {
	URI string
}

// Builder builds a new instance of the Ucfunnel adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &UcfunnelAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

func (a *UcfunnelAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	var bidReq openrtb2.BidRequest
	if err := json.Unmarshal(externalRequest.Body, &bidReq); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType := getBidType(bidReq, sb.Bid[i].ImpID)
			if (bidType == openrtb_ext.BidTypeBanner) || (bidType == openrtb_ext.BidTypeVideo) {
				b := &adapters.TypedBid{
					Bid:     &sb.Bid[i],
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
	return bidResponse, errs
}

func (a *UcfunnelAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	// If all the requests were malformed, don't bother making a server call with no impressions.
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("No impression in the bid request\n"),
		}}
	}

	partnerId, partnerErr := getPartnerId(request)
	if partnerErr != nil {
		return nil, partnerErr
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")

	uri := a.URI + "/" + url.PathEscape(partnerId) + "/request"
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     uri,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

func getPartnerId(request *openrtb2.BidRequest) (string, []error) {
	var ext ExtBidderUcfunnel
	var errs = []error{}
	err := json.Unmarshal(request.Imp[0].Ext, &ext)
	if err != nil {
		errs = append(errs, err)
		return "", errs
	}
	errs = checkBidderParameter(ext)
	if errs != nil {
		return "", errs
	}
	return ext.Bidder.PartnerId, nil
}

func checkBidderParameter(ext ExtBidderUcfunnel) []error {
	var errs = []error{}
	if len(ext.Bidder.PartnerId) == 0 || len(ext.Bidder.AdUnitId) == 0 {
		errs = append(errs, fmt.Errorf("No PartnerId or AdUnitId in the bid request\n"))
		return errs
	}
	return nil
}

func getBidType(bidReq openrtb2.BidRequest, impid string) openrtb_ext.BidType {
	for i := range bidReq.Imp {
		if bidReq.Imp[i].ID == impid {
			if bidReq.Imp[i].Banner != nil {
				return openrtb_ext.BidTypeBanner
			} else if bidReq.Imp[i].Video != nil {
				return openrtb_ext.BidTypeVideo
			} else if bidReq.Imp[i].Audio != nil {
				return openrtb_ext.BidTypeAudio
			} else if bidReq.Imp[i].Native != nil {
				return openrtb_ext.BidTypeNative
			}
		}
	}
	return openrtb_ext.BidTypeNative
}

type ExtBidderUcfunnel struct {
	Bidder openrtb_ext.ExtImpUcfunnel `json:"bidder"`
}
