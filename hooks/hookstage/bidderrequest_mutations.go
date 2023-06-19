package hookstage

import (
	"errors"

	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"
)

func (c *ChangeSet[T]) BidderRequest() ChangeSetBidderRequest[T] {
	return ChangeSetBidderRequest[T]{changeSet: c}
}

type ChangeSetBidderRequest[T any] struct {
	changeSet *ChangeSet[T]
}

func (c ChangeSetBidderRequest[T]) BAdv() ChangeSetBAdv[T] {
	return ChangeSetBAdv[T]{changeSetBidderRequest: c}
}

func (c ChangeSetBidderRequest[T]) BCat() ChangeSetBCat[T] {
	return ChangeSetBCat[T]{changeSetBidderRequest: c}
}

func (c ChangeSetBidderRequest[T]) CatTax() ChangeSetCatTax[T] {
	return ChangeSetCatTax[T]{changeSetBidderRequest: c}
}

func (c ChangeSetBidderRequest[T]) BApp() ChangeSetBApp[T] {
	return ChangeSetBApp[T]{changeSetBidderRequest: c}
}

func (c ChangeSetBidderRequest[T]) castPayload(p T) (*openrtb2.BidRequest, error) {
	if payload, ok := any(p).(BidderRequestPayload); ok {
		if payload.BidRequest == nil {
			return nil, errors.New("empty BidRequest provided")
		}
		return payload.BidRequest, nil
	}
	return nil, errors.New("failed to cast BidderRequestPayload")
}

type ChangeSetBAdv[T any] struct {
	changeSetBidderRequest ChangeSetBidderRequest[T]
}

func (c ChangeSetBAdv[T]) Update(badv []string) {
	c.changeSetBidderRequest.changeSet.AddMutation(func(p T) (T, error) {
		bidRequest, err := c.changeSetBidderRequest.castPayload(p)
		if err == nil {
			bidRequest.BAdv = badv
		}
		return p, err
	}, MutationUpdate, "bidrequest", "badv")
}

type ChangeSetBCat[T any] struct {
	changeSetBidderRequest ChangeSetBidderRequest[T]
}

func (c ChangeSetBCat[T]) Update(bcat []string) {
	c.changeSetBidderRequest.changeSet.AddMutation(func(p T) (T, error) {
		bidRequest, err := c.changeSetBidderRequest.castPayload(p)
		if err == nil {
			bidRequest.BCat = bcat
		}
		return p, err
	}, MutationUpdate, "bidrequest", "bcat")
}

type ChangeSetCatTax[T any] struct {
	changeSetBidderRequest ChangeSetBidderRequest[T]
}

func (c ChangeSetCatTax[T]) Update(cattax adcom1.CategoryTaxonomy) {
	c.changeSetBidderRequest.changeSet.AddMutation(func(p T) (T, error) {
		bidRequest, err := c.changeSetBidderRequest.castPayload(p)
		if err == nil {
			bidRequest.CatTax = cattax
		}
		return p, err
	}, MutationUpdate, "bidrequest", "cattax")
}

type ChangeSetBApp[T any] struct {
	changeSetBidderRequest ChangeSetBidderRequest[T]
}

func (c ChangeSetBApp[T]) Update(bapp []string) {
	c.changeSetBidderRequest.changeSet.AddMutation(func(p T) (T, error) {
		bidRequest, err := c.changeSetBidderRequest.castPayload(p)
		if err == nil {
			bidRequest.BApp = bapp
		}
		return p, err
	}, MutationUpdate, "bidrequest", "bapp")
}
