package openrtb_ext

// ExtImp defines the contract for bidrequest.imp[i].ext
type ExtImp struct {
	Prebid     *ExtImpPrebid     `json:"prebid"`
	Appnexus   *ExtImpAppnexus   `json:"appnexus"`
	Consumable *ExtImpConsumable `json:"consumable"`
	Rubicon    *ExtImpRubicon    `json:"rubicon"`
	Adform     *ExtImpAdform     `json:"adform"`
	Rhythmone  *ExtImpRhythmone  `json:"rhythmone"`
}

// ExtImpPrebid defines the contract for bidrequest.imp[i].ext.prebid
type ExtImpPrebid struct {
	StoredRequest *ExtStoredRequest `json:"storedrequest"`
}

// ExtStoredRequest defines the contract for bidrequest.imp[i].ext.prebid.storedrequest
type ExtStoredRequest struct {
	ID string `json:"id"`
}
