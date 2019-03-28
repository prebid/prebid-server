package openrtb_ext

import (
	"encoding/json"
)

// ExtImp defines the contract for bidrequest.imp[i].ext
type ExtImp struct {
	Prebid     *ExtImpPrebid     `json:"prebid"`
	Appnexus   *ExtImpAppnexus   `json:"appnexus"`
	Consumable *ExtImpConsumable `json:"consumable"`
	Rubicon    *ExtImpRubicon    `json:"rubicon"`
	Adform     *ExtImpAdform     `json:"adform"`
	Rhythmone  *ExtImpRhythmone  `json:"rhythmone"`
	Unruly     *ExtImpUnruly     `json:"unruly"`
}

// ExtImpPrebid defines the contract for bidrequest.imp[i].ext.prebid
type ExtImpPrebid struct {
	StoredRequest *ExtStoredRequest `json:"storedrequest"`

	// NOTE: This is not part of the official API, we are not expecting clients
	// migrate from imp[...].ext.${BIDDER} to imp[...].ext.prebid.bidder.${BIDDER}
	// at this time
	// https://github.com/prebid/prebid-server/pull/846#issuecomment-476352224
	Bidder map[string]json.RawMessage `json:"bidder"`
}

// ExtStoredRequest defines the contract for bidrequest.imp[i].ext.prebid.storedrequest
type ExtStoredRequest struct {
	ID string `json:"id"`
}
