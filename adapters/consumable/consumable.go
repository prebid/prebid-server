package consumable

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
)

type ConsumableAdapter struct{}

func (a *ConsumableAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	return nil, nil
}

func (a *ConsumableAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return nil, nil
}

func NewConsumableBidder() *ConsumableAdapter {
	return &ConsumableAdapter{}
}
