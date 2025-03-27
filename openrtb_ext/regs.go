package openrtb_ext

import "slices"

// ExtRegs defines the contract for bidrequest.regs.ext
type ExtRegs struct {
	// DSA is an object containing DSA transparency information, see https://github.com/InteractiveAdvertisingBureau/openrtb/blob/main/extensions/community_extensions/dsa_transparency.md
	DSA *ExtRegsDSA `json:"dsa,omitempty"`

	// GDPR should be "1" if the caller believes the user is subject to GDPR laws, "0" if not, and undefined
	// if it's unknown. For more info on this parameter, see: https://iabtechlab.com/wp-content/uploads/2018/02/OpenRTB_Advisory_GDPR_2018-02.pdf
	GDPR *int8 `json:"gdpr,omitempty"`

	// USPrivacy should be a four character string, see: https://iabtechlab.com/wp-content/uploads/2019/11/OpenRTB-Extension-U.S.-Privacy-IAB-Tech-Lab.pdf
	USPrivacy string `json:"us_privacy,omitempty"`
}

// ExtRegsDSA defines the contract for bidrequest.regs.ext.dsa
type ExtRegsDSA struct {
	Required     *int8                   `json:"dsarequired,omitempty"`
	PubRender    *int8                   `json:"pubrender,omitempty"`
	DataToPub    *int8                   `json:"datatopub,omitempty"`
	Transparency []ExtBidDSATransparency `json:"transparency,omitempty"`
}

// Clone creates a deep copy of ExtRegsDSA
func (erd *ExtRegsDSA) Clone() *ExtRegsDSA {
	if erd == nil {
		return nil
	}
	clone := *erd

	if erd.Required != nil {
		clonedRequired := *erd.Required
		clone.Required = &clonedRequired
	}
	if erd.PubRender != nil {
		clonedPubRender := *erd.PubRender
		clone.PubRender = &clonedPubRender
	}
	if erd.DataToPub != nil {
		clonedDataToPub := *erd.DataToPub
		clone.DataToPub = &clonedDataToPub
	}
	if erd.Transparency != nil {
		clonedTransparency := make([]ExtBidDSATransparency, len(erd.Transparency))
		for i, transparency := range erd.Transparency {
			newTransparency := transparency
			newTransparency.Params = slices.Clone(transparency.Params)
			clonedTransparency[i] = newTransparency
		}
		clone.Transparency = clonedTransparency
	}
	return &clone
}

// ExtBidDSATransparency defines the contract for bidrequest.regs.ext.dsa.transparency
type ExtBidDSATransparency struct {
	Domain string `json:"domain,omitempty"`
	Params []int  `json:"dsaparams,omitempty"`
}
