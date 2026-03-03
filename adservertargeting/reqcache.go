package adservertargeting

import (
	"encoding/json"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type requestCache struct {
	resolvedReq json.RawMessage
	impsData    []json.RawMessage
}

func (reqImpCache *requestCache) GetReqJson() []byte {
	return reqImpCache.resolvedReq
}

func (reqImpCache *requestCache) GetImpsData() ([]json.RawMessage, error) {
	if len(reqImpCache.impsData) == 0 {
		imps, _, _, err := jsonparser.Get(reqImpCache.resolvedReq, "imp")
		if err != nil {
			return nil, err
		}
		var impsData []json.RawMessage

		err = jsonutil.Unmarshal(imps, &impsData)
		if err != nil {
			return nil, err
		}
		reqImpCache.impsData = impsData
	}
	return reqImpCache.impsData, nil
}

type bidsCache struct {
	// bidder name is another layer to avoid collisions in case bid ids from different bidders will be the same
	// map[bidder name] map [bid id] bid data
	bids map[string]map[string][]byte
}

func (bidsCache *bidsCache) GetBid(bidderName, bidId string, bid openrtb2.Bid) ([]byte, error) {

	_, seatBidExists := bidsCache.bids[bidderName]
	if !seatBidExists {
		impToBid := make(map[string][]byte, 0)
		bidsCache.bids[bidderName] = impToBid
	}
	_, bidExists := bidsCache.bids[bidderName][bidId]
	if !bidExists {
		bidBytes, err := jsonutil.Marshal(bid)
		if err != nil {
			return nil, err
		}
		bidsCache.bids[bidderName][bidId] = bidBytes
	}
	return bidsCache.bids[bidderName][bidId], nil
}
