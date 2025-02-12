package openrtb_ext

import (
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
)

func ConvertUpTo26(r *RequestWrapper) error {
	if err := convertUpEnsureExt(r); err != nil {
		return err
	}

	// schain
	moveSupplyChainFrom24To25(r)
	moveSupplyChainFrom25To26(r)

	// gdpr
	moveGDPRFrom25To26(r)
	moveConsentFrom25To26(r)

	// ccpa
	moveUSPrivacyFrom25To26(r)

	// eid
	moveEIDFrom25To26(r)

	// imp
	for _, imp := range r.GetImp() {
		moveRewardedFromPrebidExtTo26(imp)
	}

	return nil
}

// convertUpEnsureExt gets all extension objects required for migration to verify there
// are no access errors.
func convertUpEnsureExt(r *RequestWrapper) error {
	if _, err := r.GetRequestExt(); err != nil {
		return fmt.Errorf("req.ext is invalid: %v", err)
	}

	if _, err := r.GetSourceExt(); err != nil {
		return fmt.Errorf("req.source.ext is invalid: %v", err)
	}

	if _, err := r.GetRegExt(); err != nil {
		return fmt.Errorf("req.regs.ext is invalid: %v", err)
	}

	if _, err := r.GetUserExt(); err != nil {
		return fmt.Errorf("req.user.ext is invalid: %v", err)
	}

	for i, imp := range r.GetImp() {
		if _, err := imp.GetImpExt(); err != nil {
			return fmt.Errorf("imp[%v].imp.ext is invalid: %v", i, err)
		}
	}

	return nil
}

// moveSupplyChainFrom24To25 modifies the request to move the OpenRTB 2.4 supply chain
// object (req.ext.schain) to the OpenRTB 2.5 location (req.source.ext.schain). If the
// OpenRTB 2.5 location is already present the OpenRTB 2.4 supply chain object is dropped.
func moveSupplyChainFrom24To25(r *RequestWrapper) {
	// read and clear 2.4 location
	reqExt, _ := r.GetRequestExt()
	schain24 := reqExt.GetSChain()
	reqExt.SetSChain(nil)

	// move to 2.5 location if not already present
	sourceExt, _ := r.GetSourceExt()
	if sourceExt.GetSChain() == nil {
		sourceExt.SetSChain(schain24)
	}
}

// moveSupplyChainFrom25To26 modifies the request to move the OpenRTB 2.5 supply chain
// object (req.source.ext.schain) to the OpenRTB 2.6 location (req.source.schain). If the
// OpenRTB 2.6 location is already present the OpenRTB 2.5 supply chain object is dropped.
func moveSupplyChainFrom25To26(r *RequestWrapper) {
	// read and clear 2.5 location
	sourceExt, _ := r.GetSourceExt()
	schain25 := sourceExt.GetSChain()
	sourceExt.SetSChain(nil)

	// move to 2.6 location if not already present
	if schain25 != nil {
		// source may be nil if moved indirectly from an OpenRTB 2.4 location, since the ext
		// is not defined on the source object.
		if r.Source == nil {
			r.Source = &openrtb2.Source{}
		}

		if r.Source.SChain == nil {
			r.Source.SChain = schain25
		}
	}
}

// moveGDPRFrom25To26 modifies the request to move the OpenRTB 2.5 GDPR signal field
// (req.regs.ext.gdpr) to the OpenRTB 2.6 location (req.regs.gdpr). If the OpenRTB 2.6
// location is already present the OpenRTB 2.5 GDPR signal is dropped.
func moveGDPRFrom25To26(r *RequestWrapper) {
	// read and clear 2.5 location
	regsExt, _ := r.GetRegExt()
	gdpr25 := regsExt.GetGDPR()
	regsExt.SetGDPR(nil)

	// move to 2.6 location
	if gdpr25 != nil && r.Regs.GDPR == nil {
		r.Regs.GDPR = gdpr25
	}
}

// moveConsentFrom25To26 modifies the request to move the OpenRTB 2.5 GDPR consent field
// (req.user.ext.consent) to the OpenRTB 2.6 location (req.user.consent). If the OpenRTB 2.6
// location is already present the OpenRTB 2.5 GDPR consent is dropped.
func moveConsentFrom25To26(r *RequestWrapper) {
	// read and clear 2.5 location
	userExt, _ := r.GetUserExt()
	consent25 := userExt.GetConsent()
	userExt.SetConsent(nil)

	// move to 2.6 location
	if consent25 != nil && r.User.Consent == "" {
		r.User.Consent = *consent25
	}
}

// moveUSPrivacyFrom25To26 modifies the request to move the OpenRTB 2.5 US Privacy (CCPA)
// consent string (req.regs.ext.usprivacy) to the OpenRTB 2.6 location (req.regs.usprivacy).
// If the OpenRTB 2.6 location is already present the OpenRTB 2.5 consent string is dropped.
func moveUSPrivacyFrom25To26(r *RequestWrapper) {
	// read and clear 2.5 location
	regsExt, _ := r.GetRegExt()
	usPrivacy25 := regsExt.GetUSPrivacy()
	regsExt.SetUSPrivacy("")

	// move to 2.6 location
	if usPrivacy25 != "" && r.Regs.USPrivacy == "" {
		r.Regs.USPrivacy = usPrivacy25
	}
}

// moveEIDFrom25To26 modifies the request to move the OpenRTB 2.5 external identifiers
// (req.user.ext.eids) to the OpenRTB 2.6 location (req.user.eids). If the OpenRTB 2.6
// location is already present the OpenRTB 2.5 external identifiers is dropped.
func moveEIDFrom25To26(r *RequestWrapper) {
	// read and clear 2.5 location
	userExt, _ := r.GetUserExt()
	eid25 := userExt.GetEid()
	userExt.SetEid(nil)

	// move to 2.6 location
	if eid25 != nil && r.User.EIDs == nil {
		r.User.EIDs = *eid25
	}
}

// moveRewardedFromPrebidExtTo26 modifies the impression to move the Prebid specific
// rewarded video signal (imp[].ext.prebid.is_rewarded_inventory) to the OpenRTB 2.6
// location (imp[].rwdd). If the OpenRTB 2.6 location is already present the Prebid
// specific extension is dropped.
func moveRewardedFromPrebidExtTo26(i *ImpWrapper) {
	// read and clear prebid ext
	impExt, _ := i.GetImpExt()
	rwddPrebidExt := (*int8)(nil)
	if p := impExt.GetPrebid(); p != nil && p.IsRewardedInventory != nil {
		rwddPrebidExt = p.IsRewardedInventory
		p.IsRewardedInventory = nil
		impExt.SetPrebid(p)
	}

	// move to 2.6 location
	if rwddPrebidExt != nil && i.Rwdd == 0 {
		i.Rwdd = *rwddPrebidExt
	}
}
