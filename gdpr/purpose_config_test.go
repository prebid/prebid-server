package gdpr

import (
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestPurposeConfigBasicEnforcementVendor(t *testing.T) {
	var (
		appnexus = string(openrtb_ext.BidderAppnexus)
		ix       = string(openrtb_ext.BidderIx)
		pubmatic = string(openrtb_ext.BidderPubmatic)
		rubicon  = string(openrtb_ext.BidderRubicon)
	)

	tests := []struct {
		description      string
		giveBasicVendors map[string]struct{}
		giveBidder       string
		wantFound        bool
	}{
		{
			description:      "vendor map is nil",
			giveBasicVendors: nil,
			giveBidder:       appnexus,
			wantFound:        false,
		},
		{
			description:      "vendor map is empty",
			giveBasicVendors: map[string]struct{}{},
			giveBidder:       appnexus,
			wantFound:        false,
		},
		{
			description: "vendor map has one bidders - bidder not found",
			giveBasicVendors: map[string]struct{}{
				pubmatic: {},
			},
			giveBidder: appnexus,
			wantFound:  false,
		},
		{
			description: "vendor map has one bidders - bidder found",
			giveBasicVendors: map[string]struct{}{
				appnexus: {},
			},
			giveBidder: appnexus,
			wantFound:  true,
		},
		{
			description: "vendor map has many bidderss - bidder not found",
			giveBasicVendors: map[string]struct{}{
				ix:       {},
				pubmatic: {},
				rubicon:  {},
			},
			giveBidder: appnexus,
			wantFound:  false,
		},
		{
			description: "vendor map has many bidderss - bidder found",
			giveBasicVendors: map[string]struct{}{
				ix:       {},
				pubmatic: {},
				appnexus: {},
				rubicon:  {},
			},
			giveBidder: appnexus,
			wantFound:  true,
		},
	}

	for _, tt := range tests {
		cfg := purposeConfig{
			BasicEnforcementVendorsMap: tt.giveBasicVendors,
		}
		found := cfg.basicEnforcementVendor(tt.giveBidder)

		assert.Equal(t, tt.wantFound, found, tt.description)
	}
}

func TestPurposeConfigVendorException(t *testing.T) {
	var (
		appnexus = string(openrtb_ext.BidderAppnexus)
		ix       = string(openrtb_ext.BidderIx)
		pubmatic = string(openrtb_ext.BidderPubmatic)
		rubicon  = string(openrtb_ext.BidderRubicon)
	)

	tests := []struct {
		description    string
		giveExceptions map[string]struct{}
		giveBidder     string
		wantFound      bool
	}{
		{
			description:    "vendor exception map is nil",
			giveExceptions: nil,
			giveBidder:     appnexus,
			wantFound:      false,
		},
		{
			description:    "vendor exception map is empty",
			giveExceptions: map[string]struct{}{},
			giveBidder:     appnexus,
			wantFound:      false,
		},
		{
			description: "vendor exception map has one bidders - bidder not found",
			giveExceptions: map[string]struct{}{
				pubmatic: {},
			},
			giveBidder: appnexus,
			wantFound:  false,
		},
		{
			description: "vendor exception map has one bidders - bidder found",
			giveExceptions: map[string]struct{}{
				appnexus: {},
			},
			giveBidder: appnexus,
			wantFound:  true,
		},
		{
			description: "vendor exception map has many bidderss - bidder not found",
			giveExceptions: map[string]struct{}{
				ix:       {},
				pubmatic: {},
				rubicon:  {},
			},
			giveBidder: appnexus,
			wantFound:  false,
		},
		{
			description: "vendor exception map has many bidderss - bidder found",
			giveExceptions: map[string]struct{}{
				ix:       {},
				pubmatic: {},
				appnexus: {},
				rubicon:  {},
			},
			giveBidder: appnexus,
			wantFound:  true,
		},
	}

	for _, tt := range tests {
		cfg := purposeConfig{
			VendorExceptionMap: tt.giveExceptions,
		}
		found := cfg.vendorException(tt.giveBidder)

		assert.Equal(t, tt.wantFound, found, tt.description)
	}
}
