package gdpr

import (
	"testing"

	"github.com/prebid/prebid-server/v2/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestPurposeConfigBasicEnforcementVendor(t *testing.T) {
	tests := []struct {
		description      string
		giveBasicVendors map[string]struct{}
		giveBidder       openrtb_ext.BidderName
		wantFound        bool
	}{
		{
			description:      "vendor map is nil",
			giveBasicVendors: nil,
			giveBidder:       openrtb_ext.BidderAppnexus,
			wantFound:        false,
		},
		{
			description:      "vendor map is empty",
			giveBasicVendors: map[string]struct{}{},
			giveBidder:       openrtb_ext.BidderAppnexus,
			wantFound:        false,
		},
		{
			description: "vendor map has one bidders - bidder not found",
			giveBasicVendors: map[string]struct{}{
				string(openrtb_ext.BidderPubmatic): {},
			},
			giveBidder: openrtb_ext.BidderAppnexus,
			wantFound:  false,
		},
		{
			description: "vendor map has one bidders - bidder found",
			giveBasicVendors: map[string]struct{}{
				string(openrtb_ext.BidderAppnexus): {},
			},
			giveBidder: openrtb_ext.BidderAppnexus,
			wantFound:  true,
		},
		{
			description: "vendor map has many bidderss - bidder not found",
			giveBasicVendors: map[string]struct{}{
				string(openrtb_ext.BidderIx):       {},
				string(openrtb_ext.BidderPubmatic): {},
				string(openrtb_ext.BidderRubicon):  {},
			},
			giveBidder: openrtb_ext.BidderAppnexus,
			wantFound:  false,
		},
		{
			description: "vendor map has many bidderss - bidder found",
			giveBasicVendors: map[string]struct{}{
				string(openrtb_ext.BidderIx):       {},
				string(openrtb_ext.BidderPubmatic): {},
				string(openrtb_ext.BidderAppnexus): {},
				string(openrtb_ext.BidderRubicon):  {},
			},
			giveBidder: openrtb_ext.BidderAppnexus,
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
	tests := []struct {
		description    string
		giveExceptions map[openrtb_ext.BidderName]struct{}
		giveBidder     openrtb_ext.BidderName
		wantFound      bool
	}{
		{
			description:    "vendor exception map is nil",
			giveExceptions: nil,
			giveBidder:     openrtb_ext.BidderAppnexus,
			wantFound:      false,
		},
		{
			description:    "vendor exception map is empty",
			giveExceptions: map[openrtb_ext.BidderName]struct{}{},
			giveBidder:     openrtb_ext.BidderAppnexus,
			wantFound:      false,
		},
		{
			description: "vendor exception map has one bidders - bidder not found",
			giveExceptions: map[openrtb_ext.BidderName]struct{}{
				openrtb_ext.BidderPubmatic: {},
			},
			giveBidder: openrtb_ext.BidderAppnexus,
			wantFound:  false,
		},
		{
			description: "vendor exception map has one bidders - bidder found",
			giveExceptions: map[openrtb_ext.BidderName]struct{}{
				openrtb_ext.BidderAppnexus: {},
			},
			giveBidder: openrtb_ext.BidderAppnexus,
			wantFound:  true,
		},
		{
			description: "vendor exception map has many bidderss - bidder not found",
			giveExceptions: map[openrtb_ext.BidderName]struct{}{
				openrtb_ext.BidderIx:       {},
				openrtb_ext.BidderPubmatic: {},
				openrtb_ext.BidderRubicon:  {},
			},
			giveBidder: openrtb_ext.BidderAppnexus,
			wantFound:  false,
		},
		{
			description: "vendor exception map has many bidderss - bidder found",
			giveExceptions: map[openrtb_ext.BidderName]struct{}{
				openrtb_ext.BidderIx:       {},
				openrtb_ext.BidderPubmatic: {},
				openrtb_ext.BidderAppnexus: {},
				openrtb_ext.BidderRubicon:  {},
			},
			giveBidder: openrtb_ext.BidderAppnexus,
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
