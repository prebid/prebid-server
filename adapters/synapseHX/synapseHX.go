package synapseHX

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}

	return bidder, nil
}

func (adapter *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData

	var tenantId string

	if len(request.Imp) == 0 {
		return nil, []error{fmt.Errorf("request contains no impressions")}
	}

	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil {
		return nil, []error{fmt.Errorf("failed to unmarshal bidder ext: %w", err)}
	}

	var ext openrtb_ext.ExtImpSynapseHX
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &ext); err != nil {
		return nil, []error{fmt.Errorf("failed to unmarshal bidder parameters: %w", err)}
	}

	tenantId = ext.TenantID

	requestBody, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to marshal request: %w", err)}
	}

	url, _ := url.Parse(adapter.endpoint)
	q := url.Query()
	q.Set("pid", tenantId)
	url.RawQuery = q.Encode()

	requestData := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     url.String(),
		Body:    requestBody,
		Headers: buildRequestHeaders(),
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	requests = append(requests, requestData)

	return requests, nil
}

func (adapter *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("failed to unmarshal response body: %v", err)}}
	}

	if len(bidResponse.SeatBid) == 0 || len(bidResponse.SeatBid[0].Bid) == 0 {
		return nil, nil
	}

	var errs []error
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResponse.SeatBid[0].Bid))

	for i := range bidResponse.SeatBid {
		for j := range bidResponse.SeatBid[i].Bid {
			bid := &bidResponse.SeatBid[i].Bid[j]
			bidType, err := getMediaTypeForBid(bid)

			if err != nil {
				errs = append(errs, err)
			} else {
				bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
					Bid:     bid,
					BidType: bidType,
				})
			}
		}
	}

	return bidderResponse, errs
}

func buildRequestHeaders() http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.6")

	return headers
}

func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.MType != 0 {
		switch bid.MType {
		case openrtb2.MarkupBanner:
			return openrtb_ext.BidTypeBanner, nil
		case openrtb2.MarkupVideo:
			return openrtb_ext.BidTypeVideo, nil
		default:
			return "", fmt.Errorf("unsupported media type %d", bid.MType)
		}
	}

	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := jsonutil.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			switch bidExt.Prebid.Type {
			case openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo:
				return bidExt.Prebid.Type, nil
			default:
				return "", fmt.Errorf("unsupported media type \"%s\"", bidExt.Prebid.Type)
			}
		}
	}

	return "", fmt.Errorf("failed to parse impression \"%s\" mediatype", bid.ImpID)
}
