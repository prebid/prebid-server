package tracing

import "time"

// testTaskPartnerID is the account resolved for testTask/01-bid-request-example.json
// (site.publisher.ext.prebid.parentAccount), used for manual end-to-end verification.
const testTaskPartnerID = "664-025-677-881"

// Rules holds the hardcoded tracing rules for the module, per the task requirement that
// tracing parameters be hardcoded rather than driven by account-level config.
//
// A trace is initiated whenever an incoming BidRequest's resolved Account.ID matches a
// rule's PartnerID. Edit this slice to change which accounts are traced and their limits.
var Rules = []Rule{
	{
		PartnerID:          testTaskPartnerID,
		Duration:           5 * time.Minute,
		TracePacketsAmount: 20,
	},
}
