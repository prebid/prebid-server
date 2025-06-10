package hookstage

import (
	"encoding/json"
	"errors"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func (c *ChangeSet[T]) ProcessedAuctionRequest() ChangeSetProcessedAuctionRequest[T] {
	return ChangeSetProcessedAuctionRequest[T]{changeSet: c}
}

type ChangeSetProcessedAuctionRequest[T any] struct {
	changeSet *ChangeSet[T]
}

func (c ChangeSetProcessedAuctionRequest[T]) Bidders() ChangeBidders[T] {
	return ChangeBidders[T]{changeSetProcessedAuctionRequest: c}
}

type ChangeBidders[T any] struct {
	changeSetProcessedAuctionRequest ChangeSetProcessedAuctionRequest[T]
}

func (c ChangeSetProcessedAuctionRequest[T]) castPayload(p T) (*openrtb_ext.RequestWrapper, error) {
	if payload, ok := any(p).(ProcessedAuctionRequestPayload); ok {
		if payload.Request == nil || payload.Request.BidRequest == nil {
			return nil, errors.New("payload contains a nil bid request")
		}
		return payload.Request, nil
	}
	return nil, errors.New("failed to cast ProcessedAuctionRequestPayload")
}

func (c ChangeBidders[T]) Update(impIdToBidders map[string]map[string]json.RawMessage) {
	c.changeSetProcessedAuctionRequest.changeSet.AddMutation(func(p T) (T, error) {
		bidRequest, err := c.changeSetProcessedAuctionRequest.castPayload(p)
		if err == nil {
			for _, impWrapper := range bidRequest.GetImp() {
				if impBidders, ok := impIdToBidders[impWrapper.ID]; ok {
					impExt, impExtErr := impWrapper.GetImpExt()
					if err != nil {
						return p, impExtErr
					}
					impPrebid := impExt.GetPrebid()
					if impPrebid == nil {
						impPrebid = &openrtb_ext.ExtImpPrebid{}
					}
					impPrebid.Bidder = impBidders
					impExt.SetPrebid(impPrebid)
				}
			}
		}
		return p, err
	}, MutationUpdate, "bidrequest", "imp", "ext", "prebid", "bidders")
}
