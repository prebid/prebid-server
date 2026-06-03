package openrtb_ext

// ExtImpTeal carries the per-imp Teal bidder-slot params.
//
// Account is required. The first valid imp's account is propagated to
// BidRequest.Site.Publisher.ID (and BidRequest.App.Publisher.ID, when present).
// All imps in a request must use the same account: a later imp whose account
// differs is rejected with a BadInput error and dropped. (Java instead silently
// keeps the first account — see the adapter's upstream Java alignment notes.)
//
// Placement is optional. When present and non-blank, the adapter injects
// imp.ext.prebid.storedrequest.id = placement on a per-imp basis. Pointer-typed
// to distinguish absent (nil) from present-empty/blank (non-nil → triggers
// validation failure mirroring Java's `placement != null && isBlank(placement)`
// check).
type ExtImpTeal struct {
	Account   string  `json:"account"`
	Placement *string `json:"placement,omitempty"`
}
