package adservertargeting

import (
	"encoding/json"
	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v17/openrtb2"
)

type reqImpCache struct {
	resolverReq json.RawMessage
	impsData    []json.RawMessage
}

func (dh *reqImpCache) GetReqJson() []byte {
	return dh.resolverReq
}

func (dh *reqImpCache) GetImpsData() ([]json.RawMessage, error) {
	if len(dh.impsData) == 0 {
		imps, _, _, err := jsonparser.Get(dh.resolverReq, "imp")
		if err != nil {
			return nil, err
		}
		var impsData []json.RawMessage

		err = json.Unmarshal(imps, &impsData)
		if err != nil {
			return nil, err
		}
		dh.impsData = impsData
	}
	return dh.impsData, nil
}

type bidsCache struct {
	//bidder name to bid id to bid data
	// bidder name is another layer to avoid collisions in case bid ids from different bidders will be the same
	bids map[string]map[string][]byte
}

func (bdh *bidsCache) GetBid(bidderName, bidId string, bid openrtb2.Bid) ([]byte, error) {

	_, seatBidExists := bdh.bids[bidderName]
	if !seatBidExists {
		impToBid := make(map[string][]byte, 0)
		bdh.bids[bidderName] = impToBid
	}
	_, biddExists := bdh.bids[bidderName][bidId]
	if !biddExists {
		bidBytes, err := json.Marshal(bid)
		if err != nil {
			return nil, err
		}
		bdh.bids[bidderName][bidId] = bidBytes
	}
	return bdh.bids[bidderName][bidId], nil
}
