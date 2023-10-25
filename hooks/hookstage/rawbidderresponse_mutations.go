package hookstage

import (
	"errors"

	"github.com/prebid/prebid-server/v2/adapters"
)

func (c *ChangeSet[T]) RawBidderResponse() ChangeSetRawBidderResponse[T] {
	return ChangeSetRawBidderResponse[T]{changeSet: c}
}

type ChangeSetRawBidderResponse[T any] struct {
	changeSet *ChangeSet[T]
}

func (c ChangeSetRawBidderResponse[T]) Bids() ChangeSetBids[T] {
	return ChangeSetBids[T]{changeSetRawBidderResponse: c}
}

func (c ChangeSetRawBidderResponse[T]) castPayload(p T) (RawBidderResponsePayload, error) {
	if payload, ok := any(p).(RawBidderResponsePayload); ok {
		return payload, nil
	}
	return RawBidderResponsePayload{}, errors.New("failed to cast RawBidderResponsePayload")
}

type ChangeSetBids[T any] struct {
	changeSetRawBidderResponse ChangeSetRawBidderResponse[T]
}

func (c ChangeSetBids[T]) Update(bids []*adapters.TypedBid) {
	c.changeSetRawBidderResponse.changeSet.AddMutation(func(p T) (T, error) {
		bidderPayload, err := c.changeSetRawBidderResponse.castPayload(p)
		if err == nil {
			bidderPayload.Bids = bids
		}
		if payload, ok := any(bidderPayload).(T); ok {
			return payload, nil
		}
		return p, errors.New("failed to cast RawBidderResponsePayload")
	}, MutationUpdate, "bids")
}
