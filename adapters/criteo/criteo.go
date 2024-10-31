package criteo

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
	endpoint   string
	bidderName string
}

type BidExt struct {
	Prebid ExtPrebid `json:"prebid"`
}

type ExtPrebid struct {
	BidType     openrtb_ext.BidType `json:"type"`
	NetworkName string              `json:"networkName"`
}

type CriteoExt struct {
	Igi []*CriteoExtIgi `json:"igi"`
}

type CriteoExtIgi struct {
	ImpId string          `json:"impid"`
	Igs   []*CriteoExtIgs `json:"igs"`
}

type CriteoExtIgs struct {
	Config json.RawMessage `json:"config"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint:   config.Endpoint,
		bidderName: string(bidderName),
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponse()
	bidResponse.Currency = response.Cur

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			var bidExt BidExt
			if err := jsonutil.Unmarshal(seatBid.Bid[i].Ext, &bidExt); err != nil {
				return nil, []error{&errortypes.BadServerResponse{
					Message: fmt.Sprintf("Missing ext.prebid.type in bid for impression : %s.", seatBid.Bid[i].ImpID),
				}}
			}

			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidExt.Prebid.BidType,
				BidMeta: getBidMeta(bidExt),
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	bidResponse.FledgeAuctionConfigs = a.ParseFledgeAuctionConfigs(response)

	return bidResponse, nil
}

func (a *adapter) ParseFledgeAuctionConfigs(response openrtb2.BidResponse) []*openrtb_ext.FledgeAuctionConfig {
	var responseExt CriteoExt
	if response.Ext != nil {
		if err := jsonutil.Unmarshal(response.Ext, &responseExt); err == nil && len(responseExt.Igi) > 0 {
			fledgeAuctionConfigs := make([]*openrtb_ext.FledgeAuctionConfig, 0, len(responseExt.Igi))
			for _, igi := range responseExt.Igi {
				if len(igi.Igs) > 0 && igi.Igs[0].Config != nil {
					fledgeAuctionConfig := &openrtb_ext.FledgeAuctionConfig{
						ImpId:  igi.ImpId,
						Bidder: a.bidderName,
						Config: igi.Igs[0].Config,
					}
					fledgeAuctionConfigs = append(fledgeAuctionConfigs, fledgeAuctionConfig)
				}
			}

			if len(fledgeAuctionConfigs) > 0 {
				return fledgeAuctionConfigs
			}
		}
	}

	return nil
}

func getBidMeta(ext BidExt) *openrtb_ext.ExtBidPrebidMeta {
	var bidMeta *openrtb_ext.ExtBidPrebidMeta
	if ext.Prebid.NetworkName != "" {
		bidMeta = &openrtb_ext.ExtBidPrebidMeta{
			NetworkName: ext.Prebid.NetworkName,
		}
	}
	return bidMeta
}
