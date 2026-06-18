package doohcreativeapproval

import (
	"testing"

	"github.com/prebid/prebid-server/v4/exchange/entities"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterResponsesByApproval(t *testing.T) {
	cfg := testModuleConfig()
	cfg.ExemptBidders = []string{"house"}
	accountID := "acct"
	appnexus := openrtb_ext.BidderName("appnexus")
	rubicon := openrtb_ext.BidderName("rubicon")
	house := openrtb_ext.BidderName("house")

	approvedBid := testBid("approved")
	rejectedBid := testBid("rejected")
	pendingBid := testBid("pending")
	unknownBid := testBid("unknown")
	exemptBid := testBid("exempt")

	responses := map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
		appnexus: {Bids: []*entities.PbsOrtbBid{approvedBid, rejectedBid, pendingBid, unknownBid}},
		rubicon:  {Bids: []*entities.PbsOrtbBid{testBid("seat-removed")}},
		house:    {Bids: []*entities.PbsOrtbBid{exemptBid}},
	}
	statuses := map[string]approvalStatus{
		creativeApprovalID(accountID, appnexus, "approved"): approvalStatusApproved,
		creativeApprovalID(accountID, appnexus, "rejected"): approvalStatusRejected,
		creativeApprovalID(accountID, appnexus, "pending"):  approvalStatusPending,
	}

	removed := filterResponsesByApproval(responses, cfg, accountID, statuses)

	assert.Equal(t, 4, removed)
	require.Contains(t, responses, appnexus)
	assert.Equal(t, []*entities.PbsOrtbBid{approvedBid}, responses[appnexus].Bids)
	assert.NotContains(t, responses, rubicon)
	require.Contains(t, responses, house)
	assert.Equal(t, []*entities.PbsOrtbBid{exemptBid}, responses[house].Bids)
}

func TestNeedsApprovalFilter(t *testing.T) {
	cfg := testModuleConfig()
	accountID := "acct"
	bidder := openrtb_ext.BidderName("appnexus")
	responses := testResponses(bidder, testBid("approved"), testBid("rejected"))
	statuses := map[string]approvalStatus{
		creativeApprovalID(accountID, bidder, "approved"): approvalStatusApproved,
		creativeApprovalID(accountID, bidder, "rejected"): approvalStatusRejected,
	}

	assert.True(t, needsApprovalFilter(responses, cfg, accountID, statuses))

	statuses[creativeApprovalID(accountID, bidder, "rejected")] = approvalStatusApproved
	assert.False(t, needsApprovalFilter(responses, cfg, accountID, statuses))
}

func TestCollectCreativeApprovalsDedupesAndSkipsExemptBidders(t *testing.T) {
	cfg := testModuleConfig()
	cfg.ExemptBidders = []string{"house"}
	accountID := "acct"
	appnexus := openrtb_ext.BidderName("appnexus")
	house := openrtb_ext.BidderName("house")

	responses := map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
		appnexus: {Bids: []*entities.PbsOrtbBid{testBid("same"), testBid("same"), testBid("")}},
		house:    {Bids: []*entities.PbsOrtbBid{testBid("exempt")}},
	}

	creatives, warnings := collectCreativeApprovals(responses, cfg, accountID)

	require.Len(t, creatives, 1)
	_, ok := creatives[creativeApprovalID(accountID, appnexus, "same")]
	assert.True(t, ok)
	assert.Len(t, warnings, 1)
}
