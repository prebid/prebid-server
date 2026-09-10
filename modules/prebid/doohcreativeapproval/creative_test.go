package doohcreativeapproval

import (
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreativeApprovalID(t *testing.T) {
	bidder := openrtb_ext.BidderName("appnexus")

	id := creativeApprovalID("acct", bidder, "cr-123")

	assert.True(t, strings.HasPrefix(id, creativeApprovalIDVersion))
	assert.Equal(t, id, creativeApprovalID("acct", bidder, "cr-123"))
	assert.NotEqual(t, id, creativeApprovalID("acct-2", bidder, "cr-123"))
	assert.NotEqual(t, id, creativeApprovalID("acct", openrtb_ext.BidderName("rubicon"), "cr-123"))
	assert.NotEqual(t, id, creativeApprovalID("acct", bidder, "cr-456"))
}

func TestNewCreativeApproval(t *testing.T) {
	bid := testBid("cr-123")
	bid.Bid.CatTax = adcom1.CategoryTaxonomy(6)

	creative, ok := newCreativeApproval("acct", openrtb_ext.BidderName("appnexus"), bid)

	require.True(t, ok)
	assert.Equal(t, creativeApprovalID("acct", openrtb_ext.BidderName("appnexus"), "cr-123"), creative.CreativeApprovalID)
	assert.Equal(t, "appnexus", creative.Bidder)
	assert.Equal(t, "cr-123", creative.CreativeID)
	assert.Equal(t, "ad-cr-123", creative.AdID)
	assert.Equal(t, "campaign-cr-123", creative.CampaignID)
	assert.Equal(t, []string{"advertiser.example"}, creative.AdvertiserDomains)
	assert.Equal(t, []string{"IAB1"}, creative.Categories)
	assert.Equal(t, 6, creative.CategoryTaxonomy)
	assert.Equal(t, "video", creative.MediaType)
	assert.EqualValues(t, 1920, creative.Width)
	assert.EqualValues(t, 1080, creative.Height)
	assert.EqualValues(t, 15, creative.Duration)
	assert.Equal(t, "deal-cr-123", creative.DealID)
	assert.Equal(t, "https://example.com/cr-123.jpg", creative.IURL)

	bid.Bid.ADomain[0] = "changed.example"
	bid.Bid.Cat[0] = "IAB2"
	assert.Equal(t, []string{"advertiser.example"}, creative.AdvertiserDomains)
	assert.Equal(t, []string{"IAB1"}, creative.Categories)
}

func TestNewCreativeApprovalMissingCreativeID(t *testing.T) {
	_, ok := newCreativeApproval("acct", openrtb_ext.BidderName("appnexus"), testBid(""))

	assert.False(t, ok)
}
