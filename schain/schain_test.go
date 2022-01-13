package schain

import (
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestBidderToPrebidChains(t *testing.T) {
	input := openrtb_ext.ExtRequest{
		Prebid: openrtb_ext.ExtRequestPrebid{
			SChains: []*openrtb_ext.ExtRequestPrebidSChain{
				{
					Bidders: []string{"Bidder1", "Bidder2"},
					SChain: openrtb_ext.ExtRequestPrebidSChainSChain{
						Complete: 1,
						Nodes: []*openrtb_ext.ExtRequestPrebidSChainSChainNode{
							{
								ASI:    "asi1",
								SID:    "sid1",
								Name:   "name1",
								RID:    "rid1",
								Domain: "domain1",
								HP:     1,
							},
							{
								ASI:    "asi2",
								SID:    "sid2",
								Name:   "name2",
								RID:    "rid2",
								Domain: "domain2",
								HP:     2,
							},
						},
						Ver: "version1",
					},
				},
				{
					Bidders: []string{"Bidder3", "Bidder4"},
					SChain:  openrtb_ext.ExtRequestPrebidSChainSChain{},
				},
			},
		},
	}

	output, err := BidderToPrebidSChains(input.Prebid.SChains)

	assert.Nil(t, err)
	assert.Equal(t, len(output), 4)
	assert.Same(t, output["Bidder1"], &input.Prebid.SChains[0].SChain)
	assert.Same(t, output["Bidder2"], &input.Prebid.SChains[0].SChain)
	assert.Same(t, output["Bidder3"], &input.Prebid.SChains[1].SChain)
	assert.Same(t, output["Bidder4"], &input.Prebid.SChains[1].SChain)
}

func TestBidderToPrebidChainsDiscardMultipleChainsForBidder(t *testing.T) {
	input := openrtb_ext.ExtRequest{
		Prebid: openrtb_ext.ExtRequestPrebid{
			SChains: []*openrtb_ext.ExtRequestPrebidSChain{
				{
					Bidders: []string{"Bidder1"},
					SChain:  openrtb_ext.ExtRequestPrebidSChainSChain{},
				},
				{
					Bidders: []string{"Bidder1", "Bidder2"},
					SChain:  openrtb_ext.ExtRequestPrebidSChainSChain{},
				},
			},
		},
	}

	output, err := BidderToPrebidSChains(input.Prebid.SChains)

	assert.NotNil(t, err)
	assert.Nil(t, output)
}

func TestBidderToPrebidChainsNilSChains(t *testing.T) {
	input := openrtb_ext.ExtRequest{
		Prebid: openrtb_ext.ExtRequestPrebid{
			SChains: nil,
		},
	}

	output, err := BidderToPrebidSChains(input.Prebid.SChains)

	assert.Nil(t, err)
	assert.Equal(t, len(output), 0)
}

func TestBidderToPrebidChainsZeroLengthSChains(t *testing.T) {
	input := openrtb_ext.ExtRequest{
		Prebid: openrtb_ext.ExtRequestPrebid{
			SChains: []*openrtb_ext.ExtRequestPrebidSChain{},
		},
	}

	output, err := BidderToPrebidSChains(input.Prebid.SChains)

	assert.Nil(t, err)
	assert.Equal(t, len(output), 0)
}
