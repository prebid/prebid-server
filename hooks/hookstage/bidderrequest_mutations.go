package hookstage

import (
	"encoding/json"
	"errors"

	"github.com/prebid/prebid-server/v3/openrtb_ext"

	"github.com/prebid/openrtb/v20/adcom1"
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

func (c ChangeSetBidderRequest[T]) Bidders() ChangeBidders[T] {
	return ChangeBidders[T]{changeSetBidderRequest: c}
}

func (c ChangeSetBidderRequest[T]) castPayload(p T) (*openrtb_ext.RequestWrapper, error) {
	if payload, ok := any(p).(BidderRequestPayload); ok {
		if payload.Request == nil || payload.Request.BidRequest == nil {
			return nil, errors.New("payload contains a nil bid request")
		}
		return payload.Request, nil
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

type ChangeBidders[T any] struct {
	changeSetBidderRequest ChangeSetBidderRequest[T]
}

func (c ChangeBidders[T]) Add(bidders []string) {
	c.changeSetBidderRequest.changeSet.AddMutation(func(p T) (T, error) {
		bidRequest, err := c.changeSetBidderRequest.castPayload(p)
		if err == nil {
			for _, impWrapper := range bidRequest.GetImp() {

				impExt, impExtErr := impWrapper.GetImpExt()
				if err != nil {
					return p, impExtErr
				}
				impPrebid := impExt.GetPrebid()
				impBidders := impPrebid.Bidder

				newBidders := make(map[string]json.RawMessage, 0)

				for _, bidder := range bidders {
					if bidderParams, ok := impBidders[bidder]; ok {
						// keep only bidders that are present in bidders []string
						newBidders[bidder] = bidderParams
					}
				}
				impPrebid.Bidder = newBidders
			}
		}
		return p, err
	}, MutationAdd, "bidrequest", "imp", "ext", "prebid", "bidders")
}

func (c ChangeBidders[T]) Delete(bidders []string) {
	c.changeSetBidderRequest.changeSet.AddMutation(func(p T) (T, error) {
		bidRequest, err := c.changeSetBidderRequest.castPayload(p)
		if err == nil {
			if err == nil {
				for _, impWrapper := range bidRequest.GetImp() {

					impExt, impExtErr := impWrapper.GetImpExt()
					if err != nil {
						return p, impExtErr
					}
					impPrebid := impExt.GetPrebid()
					impBidders := impPrebid.Bidder

					newBidders := make(map[string]json.RawMessage, 0)

					for _, bidder := range bidders {
						if bidderParams, ok := impBidders[bidder]; !ok {
							// remove bidders that are present in bidders []string
							newBidders[bidder] = bidderParams
						}
					}
					impPrebid.Bidder = newBidders
				}
			}
		}
		return p, err
	}, MutationDelete, "bidrequest", "imp", "ext", "prebid", "bidders")
}
