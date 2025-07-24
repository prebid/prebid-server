package hookstage

import (
	"encoding/json"
	"errors"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"slices"
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

func (c ChangeBidders[T]) Add(finalBidders []string) {
	c.changeSetProcessedAuctionRequest.changeSet.AddMutation(func(p T) (T, error) {
		bidRequest, err := c.changeSetProcessedAuctionRequest.castPayload(p)
		if err == nil {
			for _, impWrapper := range bidRequest.GetImp() {

				impExt, impExtErr := impWrapper.GetImpExt()
				if err != nil {
					return p, impExtErr
				}
				impPrebid := impExt.GetPrebid()
				if impPrebid == nil {
					impPrebid = &openrtb_ext.ExtImpPrebid{}
				}
				resBiddders := make(map[string]json.RawMessage)
				for impBidder, bidderParams := range impPrebid.Bidder {
					for _, finalBidder := range finalBidders {
						if impBidder == finalBidder {
							resBiddders[impBidder] = bidderParams
						}
					}
				}
				impPrebid.Bidder = resBiddders
				impExt.SetPrebid(impPrebid)

			}
		}
		return p, err
	}, MutationAdd, "bidrequest", "imp", "ext", "prebid", "bidders")
}

func (c ChangeBidders[T]) Delete(biddersToDelete []string) {
	c.changeSetProcessedAuctionRequest.changeSet.AddMutation(func(p T) (T, error) {
		bidRequest, err := c.changeSetProcessedAuctionRequest.castPayload(p)
		if err == nil {
			for _, impWrapper := range bidRequest.GetImp() {
				impExt, impExtErr := impWrapper.GetImpExt()
				if err != nil {
					return p, impExtErr
				}
				impPrebid := impExt.GetPrebid()
				if impPrebid == nil {
					return p, nil
				}

				newImpBidders := make(map[string]json.RawMessage)

				for bidderName, bidderData := range impPrebid.Bidder {
					if contains := slices.Contains(biddersToDelete, bidderName); !contains {
						newImpBidders[bidderName] = bidderData
					}
				}

				impPrebid.Bidder = newImpBidders
				impExt.SetPrebid(impPrebid)

			}
		}
		return p, err
	}, MutationDelete, "bidrequest", "imp", "ext", "prebid", "bidders")
}
