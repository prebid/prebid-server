package openrtb_ext

import "github.com/prebid/openrtb/v20/adcom1"

func ConvertDownTo25(r *RequestWrapper) error {
	// schain
	if err := moveSupplyChainFrom26To25(r); err != nil {
		return err
	}

	// gdpr
	if err := moveGDPRFrom26To25(r); err != nil {
		return err
	}
	if err := moveConsentFrom26To25(r); err != nil {
		return err
	}

	// ccpa
	if err := moveUSPrivacyFrom26To25(r); err != nil {
		return err
	}

	// eid
	if err := moveEIDFrom26To25(r); err != nil {
		return err
	}

	// imp
	for _, imp := range r.GetImp() {
		if err := moveRewardedFrom26ToPrebidExt(imp); err != nil {
			return err
		}
	}

	return nil
}

// moveSupplyChainFrom26To25 modifies the request to move the OpenRTB 2.6 supply chain
// object (req.source.schain) to the OpenRTB 2.5 location (req.source.ext.schain). If the
// OpenRTB 2.5 location is already present it may be overwritten. The OpenRTB 2.5 location
// is expected to be empty.
func moveSupplyChainFrom26To25(r *RequestWrapper) error {
	if r.Source == nil || r.Source.SChain == nil {
		return nil
	}

	// read and clear 2.6 location
	schain26 := r.Source.SChain
	r.Source.SChain = nil

	// move to 2.5 location
	sourceExt, err := r.GetSourceExt()
	if err != nil {
		return err
	}
	sourceExt.SetSChain(schain26)

	return nil
}

// moveGDPRFrom26To25 modifies the request to move the OpenRTB 2.6 GDPR signal
// field (req.regs.gdpr) to the OpenRTB 2.5 location (req.regs.ext.gdpr). If the
// OpenRTB 2.5 location is already present it may be overwritten. The OpenRTB 2.5
// location is expected to be empty.
func moveGDPRFrom26To25(r *RequestWrapper) error {
	if r.Regs == nil || r.Regs.GDPR == nil {
		return nil
	}

	// read and clear 2.6 location
	gdpr26 := r.Regs.GDPR
	r.Regs.GDPR = nil

	// move to 2.5 location
	regExt, err := r.GetRegExt()
	if err != nil {
		return err
	}
	regExt.SetGDPR(gdpr26)

	return nil
}

// moveConsentFrom26To25 modifies the request to move the OpenRTB 2.6 GDPR consent
// field (req.user.consent) to the OpenRTB 2.5 location (req.user.ext.consent). If
// the OpenRTB 2.5 location is already present it may be overwritten. The OpenRTB 2.5
// location is expected to be empty.
func moveConsentFrom26To25(r *RequestWrapper) error {
	if r.User == nil || len(r.User.Consent) == 0 {
		return nil
	}

	// read and clear 2.6 location
	consent26 := r.User.Consent
	r.User.Consent = ""

	// move to 2.5 location
	userExt, err := r.GetUserExt()
	if err != nil {
		return err
	}
	userExt.SetConsent(&consent26)

	return nil
}

// moveUSPrivacyFrom26To25 modifies the request to move the OpenRTB 2.6 US Privacy (CCPA)
// consent string (req.regs.us_privacy) to the OpenRTB 2.5 location (req.regs.ext.us_privacy).
// If the OpenRTB 2.5 location is already present it may be overwritten. The OpenRTB 2.5
// location is expected to be empty.
func moveUSPrivacyFrom26To25(r *RequestWrapper) error {
	if r.Regs == nil || len(r.Regs.USPrivacy) == 0 {
		return nil
	}

	// read and clear 2.6 location
	usprivacy26 := r.Regs.USPrivacy
	r.Regs.USPrivacy = ""

	// move to 2.5 location
	regExt, err := r.GetRegExt()
	if err != nil {
		return err
	}
	regExt.SetUSPrivacy(usprivacy26)

	return nil
}

// moveEIDFrom26To25 modifies the request to move the OpenRTB 2.6 external identifiers
// (req.user.eids) to the OpenRTB 2.5 location (req.user.ext.eids). If the OpenRTB 2.5
// location is already present it may be overwritten. The OpenRTB 2.5 location is
// expected to be empty.
func moveEIDFrom26To25(r *RequestWrapper) error {
	if r.User == nil || r.User.EIDs == nil {
		return nil
	}

	// read and clear 2.6 location
	eid26 := r.User.EIDs
	r.User.EIDs = nil

	// move to 2.5 location
	userExt, err := r.GetUserExt()
	if err != nil {
		return err
	}
	userExt.SetEid(&eid26)

	return nil
}

// moveRewardedFrom26ToPrebidExt modifies the impression to move the OpenRTB 2.6 rewarded
// signal (imp[].rwdd) to the OpenRTB 2.x Prebid specific location (imp[].ext.prebid.is_rewarded_inventory).
// If the Prebid specific location is already present, it may be overwritten. The Prebid specific
// location is expected to be empty.
func moveRewardedFrom26ToPrebidExt(i *ImpWrapper) error {
	if i.Rwdd == 0 {
		return nil
	}

	// read and clear 2.6 location
	rwdd26 := i.Rwdd
	i.Rwdd = 0

	// move to Prebid specific location
	impExt, err := i.GetImpExt()
	if err != nil {
		return err
	}
	impExtPrebid := impExt.GetOrCreatePrebid()
	impExtPrebid.IsRewardedInventory = &rwdd26
	impExt.SetPrebid(impExtPrebid)

	return nil
}

// Clear26Fields sets all fields introduced in OpenRTB 2.6 to default values, which
// will cause them to be omitted during json marshal.
func Clear26Fields(r *RequestWrapper) {
	r.WLangB = nil
	r.CatTax = 0

	if app := r.App; app != nil {
		app.CatTax = 0
		app.KwArray = nil

		if content := r.App.Content; content != nil {
			content.CatTax = 0
			content.KwArray = nil
			content.LangB = ""
			content.Network = nil
			content.Channel = nil

			if producer := r.App.Content.Producer; producer != nil {
				producer.CatTax = 0
			}
		}

		if publisher := r.App.Publisher; publisher != nil {
			publisher.CatTax = 0
		}
	}

	if site := r.Site; site != nil {
		site.CatTax = 0
		site.KwArray = nil

		if content := r.Site.Content; content != nil {
			content.CatTax = 0
			content.KwArray = nil
			content.LangB = ""
			content.Network = nil
			content.Channel = nil

			if producer := r.Site.Content.Producer; producer != nil {
				producer.CatTax = 0
			}
		}

		if publisher := r.Site.Publisher; publisher != nil {
			publisher.CatTax = 0
		}
	}

	if device := r.Device; device != nil {
		device.SUA = nil
		device.LangB = ""
	}

	if regs := r.Regs; regs != nil {
		regs.GDPR = nil
		regs.USPrivacy = ""
	}

	if source := r.Source; source != nil {
		source.SChain = nil
	}

	if user := r.User; user != nil {
		user.KwArray = nil
		user.Consent = ""
		user.EIDs = nil
	}

	for _, imp := range r.GetImp() {
		imp.Rwdd = 0
		imp.SSAI = 0

		if audio := imp.Audio; audio != nil {
			audio.PodDur = 0
			audio.RqdDurs = nil
			audio.PodID = ""
			audio.PodSeq = 0
			audio.SlotInPod = 0
			audio.MinCPMPerSec = 0
		}

		if video := imp.Video; video != nil {
			video.MaxSeq = 0
			video.PodDur = 0
			video.PodID = ""
			video.PodSeq = 0
			video.RqdDurs = nil
			video.SlotInPod = 0
			video.MinCPMPerSec = 0
		}
	}
}

// Clear202211Fields sets all fields introduced in OpenRTB 2.6-202211 to default values
// which will cause them to be omitted during json marshal.
func Clear202211Fields(r *RequestWrapper) {
	r.DOOH = nil

	if app := r.App; app != nil {
		app.InventoryPartnerDomain = ""
	}

	if site := r.Site; site != nil {
		site.InventoryPartnerDomain = ""
	}

	if regs := r.Regs; regs != nil {
		regs.GPP = ""
		regs.GPPSID = nil
	}

	for _, imp := range r.GetImp() {
		imp.Qty = nil
		imp.DT = 0
	}
}

// Clear202303Fields sets all fields introduced in OpenRTB 2.6-202303 to default values
// which will cause them to be omitted during json marshal.
func Clear202303Fields(r *RequestWrapper) {
	for _, imp := range r.GetImp() {
		imp.Refresh = nil

		if video := imp.Video; video != nil {
			video.Plcmt = 0
		}
	}
}

// Clear202309Fields sets all fields introduced in OpenRTB 2.6-202309 to default values
// which will cause them to be omitted during json marshal.
func Clear202309Fields(r *RequestWrapper) {
	r.ACat = nil

	for _, imp := range r.GetImp() {
		if audio := imp.Audio; audio != nil {
			audio.DurFloors = nil
		}

		if video := imp.Video; video != nil {
			video.DurFloors = nil
		}

		if pmp := imp.PMP; pmp != nil {
			for i := range pmp.Deals {
				pmp.Deals[i].Guar = 0
				pmp.Deals[i].MinCPMPerSec = 0
				pmp.Deals[i].DurFloors = nil
			}
		}
	}
}

// Clear202402Fields sets all fields introduced in OpenRTB 2.6-202402 to default values
// which will cause them to be omitted during json marshal.
func Clear202402Fields(r *RequestWrapper) {
	for _, imp := range r.GetImp() {
		if video := imp.Video; video != nil {
			video.PodDedupe = nil
		}
	}
}

// Clear202409Fields sets all fields introduced in OpenRTB 2.6-202409 to default values
// which will cause them to be omitted during json marshal.
func Clear202409Fields(r *RequestWrapper) {
	if user := r.User; user != nil {
		for i := range user.EIDs {
			user.EIDs[i].Inserter = ""
			user.EIDs[i].Matcher = ""
			user.EIDs[i].MM = adcom1.MatchMethodUnknown
		}
	}
}
