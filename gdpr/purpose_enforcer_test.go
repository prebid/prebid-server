package gdpr

import (
	"testing"

	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestNewPurposeEnforcerBuilder(t *testing.T) {
	appnexus := string(openrtb_ext.BidderAppnexus)

	tests := []struct {
		description        string
		enforceAlgo        config.TCF2EnforcementAlgo
		enforcePurpose     bool
		enforceVendors     bool
		basicVendorsMap    map[string]struct{}
		vendorExceptionMap map[string]struct{}
		purpose            consentconstants.Purpose
		bidder             string
		wantType           PurposeEnforcer
	}{
		{
			description:        "purpose 1 full algo -- full enforcer returned",
			enforceAlgo:        config.TCF2FullEnforcement,
			enforcePurpose:     true,
			enforceVendors:     true,
			basicVendorsMap:    map[string]struct{}{},
			vendorExceptionMap: map[string]struct{}{},
			purpose:            consentconstants.Purpose(1),
			bidder:             appnexus,
			wantType:           &FullEnforcement{},
		},
		{
			description:        "purpose 1 full algo, basic enforcement vendor -- full enforcer returned",
			enforceAlgo:        config.TCF2FullEnforcement,
			enforcePurpose:     true,
			enforceVendors:     true,
			basicVendorsMap:    map[string]struct{}{appnexus: {}},
			vendorExceptionMap: map[string]struct{}{},
			purpose:            consentconstants.Purpose(1),
			bidder:             appnexus,
			wantType:           &FullEnforcement{},
		},
		{
			description:        "purpose 1 basic algo -- basic enforcer returned",
			enforceAlgo:        config.TCF2BasicEnforcement,
			enforcePurpose:     true,
			enforceVendors:     true,
			basicVendorsMap:    map[string]struct{}{},
			vendorExceptionMap: map[string]struct{}{},
			purpose:            consentconstants.Purpose(1),
			bidder:             appnexus,
			wantType:           &BasicEnforcement{},
		},
		{
			description:        "purpose 2 full algo -- full enforcer returned",
			enforceAlgo:        config.TCF2FullEnforcement,
			enforcePurpose:     true,
			enforceVendors:     true,
			basicVendorsMap:    map[string]struct{}{},
			vendorExceptionMap: map[string]struct{}{},
			purpose:            consentconstants.Purpose(2),
			bidder:             appnexus,
			wantType:           &FullEnforcement{},
		},
		{
			description:        "purpose 2 full algo, basic enforcement vendor -- basic enforcer returned",
			enforceAlgo:        config.TCF2FullEnforcement,
			enforcePurpose:     true,
			enforceVendors:     true,
			basicVendorsMap:    map[string]struct{}{appnexus: {}},
			vendorExceptionMap: map[string]struct{}{},
			purpose:            consentconstants.Purpose(2),
			bidder:             appnexus,
			wantType:           &BasicEnforcement{},
		},
		{
			description:        "purpose 2 basic algo -- basic enforcer returned",
			enforceAlgo:        config.TCF2BasicEnforcement,
			enforcePurpose:     true,
			enforceVendors:     true,
			basicVendorsMap:    map[string]struct{}{},
			vendorExceptionMap: map[string]struct{}{},
			purpose:            consentconstants.Purpose(2),
			bidder:             appnexus,
			wantType:           &BasicEnforcement{},
		},
	}

	for _, tt := range tests {
		cfg := fakeTCF2ConfigReader{
			enforceAlgo:                tt.enforceAlgo,
			enforcePurpose:             tt.enforcePurpose,
			enforceVendors:             tt.enforceVendors,
			basicEnforcementVendorsMap: tt.basicVendorsMap,
			vendorExceptionMap:         tt.vendorExceptionMap,
		}

		builder := NewPurposeEnforcerBuilder(&cfg)

		enforcer1 := builder(tt.purpose, tt.bidder)
		enforcer2 := builder(tt.purpose, tt.bidder)

		// assert that enforcer1 and enforcer2 are same objects; enforcer2 pulled from cache
		assert.Same(t, enforcer1, enforcer2, tt.description)
		assert.IsType(t, tt.wantType, enforcer1, tt.description)

		// assert enforcer 1 config values are properly set
		switch enforcerCasted := enforcer1.(type) {
		case *FullEnforcement:
			{
				fullEnforcer := enforcerCasted
				assert.Equal(t, fullEnforcer.cfg.PurposeID, tt.purpose, tt.description)
				assert.Equal(t, fullEnforcer.cfg.EnforceAlgo, tt.enforceAlgo, tt.description)
				assert.Equal(t, fullEnforcer.cfg.EnforcePurpose, tt.enforcePurpose, tt.description)
				assert.Equal(t, fullEnforcer.cfg.EnforceVendors, tt.enforceVendors, tt.description)
				assert.Equal(t, fullEnforcer.cfg.BasicEnforcementVendorsMap, tt.basicVendorsMap, tt.description)
				assert.Equal(t, fullEnforcer.cfg.VendorExceptionMap, tt.vendorExceptionMap, tt.description)
			}
		case PurposeEnforcer:
			{
				basicEnforcer := enforcer1.(*BasicEnforcement)
				assert.Equal(t, basicEnforcer.cfg.PurposeID, tt.purpose, tt.description)
				assert.Equal(t, basicEnforcer.cfg.EnforceAlgo, tt.enforceAlgo, tt.description)
				assert.Equal(t, basicEnforcer.cfg.EnforcePurpose, tt.enforcePurpose, tt.description)
				assert.Equal(t, basicEnforcer.cfg.EnforceVendors, tt.enforceVendors, tt.description)
				assert.Equal(t, basicEnforcer.cfg.BasicEnforcementVendorsMap, tt.basicVendorsMap, tt.description)
				assert.Equal(t, basicEnforcer.cfg.VendorExceptionMap, tt.vendorExceptionMap, tt.description)
			}
		default:
			assert.FailNow(t, "unexpected type of enforcer")
		}
	}
}

func TestNewPurposeEnforcerBuilderCaching(t *testing.T) {

	bidder1 := string(openrtb_ext.BidderAppnexus)
	bidder1Enforcers := make([]PurposeEnforcer, 11)
	bidder2 := string(openrtb_ext.BidderIx)
	bidder2Enforcers := make([]PurposeEnforcer, 11)
	bidder3 := string(openrtb_ext.BidderPubmatic)
	bidder3Enforcers := make([]PurposeEnforcer, 11)
	bidder4 := string(openrtb_ext.BidderRubicon)
	bidder4Enforcers := make([]PurposeEnforcer, 11)

	cfg := fakeTCF2ConfigReader{
		enforceAlgo: config.TCF2FullEnforcement,
		basicEnforcementVendorsMap: map[string]struct{}{
			string(bidder3): {},
			string(bidder4): {},
		},
	}
	builder := NewPurposeEnforcerBuilder(&cfg)

	for i := 1; i <= 10; i++ {
		bidder1Enforcers[i] = builder(consentconstants.Purpose(i), bidder1)
		bidder2Enforcers[i] = builder(consentconstants.Purpose(i), bidder2)
		bidder3Enforcers[i] = builder(consentconstants.Purpose(i), bidder3)
		bidder4Enforcers[i] = builder(consentconstants.Purpose(i), bidder4)
	}

	for i := 1; i <= 10; i++ {
		if i == 1 {
			assert.IsType(t, bidder1Enforcers[i], &FullEnforcement{}, "purpose 1 bidder 1 enforcer is full")
			assert.IsType(t, bidder3Enforcers[i], &FullEnforcement{}, "purpose 1 bidder 3 enforcer is full")

			// verify cross-bidder enforcer objects for a given purpose are the same
			assert.Same(t, bidder1Enforcers[i], bidder2Enforcers[i], "purpose 1 compare bidder 1 & 2 enforcers")
			assert.Same(t, bidder2Enforcers[i], bidder3Enforcers[i], "purpose 1 compare bidder 2 & 3 enforcers")
			assert.Same(t, bidder3Enforcers[i], bidder4Enforcers[i], "purpose 1 compare bidder 3 & 4 enforcers")

			// verify cross-purpose enforcer objects are different
			assert.Equal(t, bidder1Enforcers[i].(*FullEnforcement).cfg.PurposeID, consentconstants.Purpose(i), "purpose 1 bidder 1 enforcer purpose check")
			assert.Equal(t, bidder2Enforcers[i].(*FullEnforcement).cfg.PurposeID, consentconstants.Purpose(i), "purpose 1 bidder 2 enforcer purpose check")
			assert.Equal(t, bidder3Enforcers[i].(*FullEnforcement).cfg.PurposeID, consentconstants.Purpose(i), "purpose 1 bidder 3 enforcer purpose check")
			assert.Equal(t, bidder4Enforcers[i].(*FullEnforcement).cfg.PurposeID, consentconstants.Purpose(i), "purpose 1 bidder 4 enforcer purpose check")
		} else {
			assert.IsType(t, bidder1Enforcers[i], &FullEnforcement{}, "purpose %d bidder 1 enforcer is full", i)
			assert.IsType(t, bidder3Enforcers[i], &BasicEnforcement{}, "purpose %d bidder 3 enforcer is basic", i)

			// verify some cross-bidder enforcer objects for a given purpose are the same and some are different
			assert.Same(t, bidder1Enforcers[i], bidder2Enforcers[i], "purpose %d compare bidder 1 & 2 enforcers", i)
			assert.NotSame(t, bidder2Enforcers[i], bidder3Enforcers[i], "purpose %d compare bidder 2 & 3 enforcers", i)
			assert.Same(t, bidder3Enforcers[i], bidder4Enforcers[i], "purpose %d compare bidder 3 & 4 enforcers", i)

			// verify cross-purpose enforcer objects are different
			assert.Equal(t, bidder1Enforcers[i].(*FullEnforcement).cfg.PurposeID, consentconstants.Purpose(i), "purpose %d bidder 1 enforcer purpose check", i)
			assert.Equal(t, bidder2Enforcers[i].(*FullEnforcement).cfg.PurposeID, consentconstants.Purpose(i), "purpose %d bidder 2 enforcer purpose check", i)
			assert.Equal(t, bidder3Enforcers[i].(*BasicEnforcement).cfg.PurposeID, consentconstants.Purpose(i), "purpose %d bidder 3 enforcer purpose check", i)
			assert.Equal(t, bidder4Enforcers[i].(*BasicEnforcement).cfg.PurposeID, consentconstants.Purpose(i), "purpose %d bidder 4 enforcer purpose check", i)
		}
	}
}

type fakeTCF2ConfigReader struct {
	enforceAlgo                config.TCF2EnforcementAlgo
	enforcePurpose             bool
	enforceVendors             bool
	vendorExceptionMap         map[string]struct{}
	basicEnforcementVendorsMap map[string]struct{}
}

func (fcr *fakeTCF2ConfigReader) BasicEnforcementVendors() map[string]struct{} {
	return fcr.basicEnforcementVendorsMap
}
func (fcr *fakeTCF2ConfigReader) FeatureOneEnforced() bool {
	return false
}
func (fcr *fakeTCF2ConfigReader) FeatureOneVendorException(openrtb_ext.BidderName) bool {
	return false
}
func (fcr *fakeTCF2ConfigReader) ChannelEnabled(config.ChannelType) bool {
	return false
}
func (fcr *fakeTCF2ConfigReader) IsEnabled() bool {
	return false
}
func (fcr *fakeTCF2ConfigReader) PurposeEnforced(purpose consentconstants.Purpose) bool {
	return fcr.enforcePurpose
}
func (fcr *fakeTCF2ConfigReader) PurposeEnforcementAlgo(purpose consentconstants.Purpose) config.TCF2EnforcementAlgo {
	return fcr.enforceAlgo
}
func (fcr *fakeTCF2ConfigReader) PurposeEnforcingVendors(purpose consentconstants.Purpose) bool {
	return fcr.enforceVendors
}
func (fcr *fakeTCF2ConfigReader) PurposeVendorExceptions(purpose consentconstants.Purpose) map[string]struct{} {
	return fcr.vendorExceptionMap
}
func (fcr *fakeTCF2ConfigReader) PurposeOneTreatmentEnabled() bool {
	return false
}
func (fcr *fakeTCF2ConfigReader) PurposeOneTreatmentAccessAllowed() bool {
	return false
}
