package openrtb_ext

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

	// Remove new OpenRTB 2.6 fields. The spec did not specify that bidders and exchanges
	// must tolerate new or unexpected fields gracefully until 2.6.
	clear26Fields(r)
	clear202211Fields(r)

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

// clear26Fields sets all fields introduced in OpenRTB 2.6 to their defaults, which
// will cause them to omitted during json marshal.
func clear26Fields(r *RequestWrapper) {
	r.WLangB = nil
	r.CatTax = 0

	if r.App != nil {
		r.App.CatTax = 0
		r.App.KwArray = nil

		if r.App.Content != nil {
			r.App.Content.CatTax = 0
			r.App.Content.KwArray = nil
			r.App.Content.LangB = ""
			r.App.Content.Network = nil
			r.App.Content.Channel = nil

			if r.App.Content.Producer != nil {
				r.App.Content.Producer.CatTax = 0
			}
		}

		if r.App.Publisher != nil {
			r.App.Publisher.CatTax = 0
		}
	}

	if r.Site != nil {
		r.Site.CatTax = 0
		r.Site.KwArray = nil

		if r.Site.Content != nil {
			r.Site.Content.CatTax = 0
			r.Site.Content.KwArray = nil
			r.Site.Content.LangB = ""
			r.Site.Content.Network = nil
			r.Site.Content.Channel = nil

			if r.Site.Content.Producer != nil {
				r.Site.Content.Producer.CatTax = 0
			}
		}

		if r.Site.Publisher != nil {
			r.Site.Publisher.CatTax = 0
		}
	}

	if r.Device != nil {
		r.Device.UA = ""
		r.Device.SUA = nil
		r.Device.LangB = ""
	}

	if r.Regs != nil {
		r.Regs.GDPR = nil
		r.Regs.USPrivacy = ""
	}

	if r.Source != nil {
		r.Source.SChain = nil
	}

	if r.User != nil {
		r.User.KwArray = nil
		r.User.Consent = ""
		r.User.EIDs = nil
	}

	for _, imp := range r.GetImp() {
		imp.Rwdd = 0
		imp.SSAI = 0

		if imp.Audio != nil {
			imp.Audio.PodDur = 0
			imp.Audio.RqdDurs = nil
			imp.Audio.PodID = 0
			imp.Audio.PodSeq = 0
			imp.Audio.SlotInPod = 0
			imp.Audio.MinCPMPerSec = 0
		}

		if imp.Video != nil {
			imp.Video.StartDelay = nil
			imp.Video.MaxSeq = 0
			imp.Video.PodDur = 0
			imp.Video.PodID = 0
			imp.Video.PodSeq = 0
			imp.Video.RqdDurs = nil
			imp.Video.SlotInPod = 0
			imp.Video.MinCPMPerSec = 0
		}
	}
}

// clear202211Fields sets all fields introduced in OpenRTB 2.6-202211 to their defaults,
// which will cause them to omitted during json marshal.
func clear202211Fields(r *RequestWrapper) {
	r.DOOH = nil

	if r.App != nil {
		r.App.InventoryPartnerDomain = ""
	}

	if r.Site != nil {
		r.Site.InventoryPartnerDomain = ""
	}

	if r.Regs != nil {
		r.Regs.GPP = ""
		r.Regs.GPPSID = nil
	}

	for _, imp := range r.GetImp() {
		imp.Qty = nil
		imp.DT = 0
	}
}
