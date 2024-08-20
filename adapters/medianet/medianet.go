package medianet

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

type adapter struct {
	bidderName string
	endpoint   string
}

type interestGroupAuctionBuyer struct {
	Origin          string          `json:"origin"`
	MaxBid          float64         `json:"maxbid"`
	Currency        string          `json:"cur"`
	BuyerSignals    string          `json:"pbs"`
	PrioritySignals json.RawMessage `json:"ps"`
}

type interestGroupAuctionSeller struct {
	ImpId  string          `json:"impid"`
	Config json.RawMessage `json:"config"`
}

type interestGroupIntent struct {
	Igb []interestGroupAuctionBuyer  `json:"igb,omitempty"`
	Igs []interestGroupAuctionSeller `json:"igs,omitempty"`
}

type medianetRespExt struct {
	Igi []interestGroupIntent `json:"igi,omitempty"`
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	reqJson, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJson,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, errs
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

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

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponse()

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getBidMediaTypeFromMtype(&sb.Bid[i])
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					Bid:     &sb.Bid[i],
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}

	if fledgeAuctionConfigs, err := extractFledge(a, bidResp); err == nil && fledgeAuctionConfigs != nil {
		bidResponse.FledgeAuctionConfigs = fledgeAuctionConfigs
	}

	return bidResponse, errs
}

// Builder builds a new instance of the Medianet adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	url := buildEndpoint(config.Endpoint, config.ExtraAdapterInfo)
	return &adapter{
		bidderName: string(bidderName),
		endpoint:   url,
	}, nil
}

func getBidMediaTypeFromMtype(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("Unable to fetch mediaType for imp: %s", bid.ImpID)
	}
}

func extractFledge(a *adapter, bidResp openrtb2.BidResponse) ([]*openrtb_ext.FledgeAuctionConfig, error) {
	var fledgeAuctionConfigs []*openrtb_ext.FledgeAuctionConfig

	var bidRespExt medianetRespExt
	if err := json.Unmarshal(bidResp.Ext, &bidRespExt); err != nil {
		return nil, err
	}

	for _, igi := range bidRespExt.Igi {
		for _, igs := range igi.Igs {
			if fledgeAuctionConfigs == nil {
				fledgeAuctionConfigs = make([]*openrtb_ext.FledgeAuctionConfig, 0)
			}
			fledgeConfig := &openrtb_ext.FledgeAuctionConfig{
				ImpId:  igs.ImpId,
				Bidder: a.bidderName,
				Config: igs.Config,
			}
			fledgeAuctionConfigs = append(fledgeAuctionConfigs, fledgeConfig)
		}
	}

	return fledgeAuctionConfigs, nil
}

func buildEndpoint(mnetUrl, hostUrl string) string {

	if len(hostUrl) == 0 {
		return mnetUrl
	}
	urlObject, err := url.Parse(mnetUrl)
	if err != nil {
		return mnetUrl
	}
	values := urlObject.Query()
	values.Add("src", hostUrl)
	urlObject.RawQuery = values.Encode()
	return urlObject.String()
}
