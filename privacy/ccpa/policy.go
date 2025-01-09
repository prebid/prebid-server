package ccpa

import (
	"errors"
	"fmt"

	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	gppPolicy "github.com/prebid/prebid-server/v3/privacy/gpp"
)

// Policy represents the CCPA regulatory information from an OpenRTB bid request.
type Policy struct {
	Consent       string
	NoSaleBidders []string
}

// ReadFromRequestWrapper extracts the CCPA regulatory information from an OpenRTB bid request.
func ReadFromRequestWrapper(req *openrtb_ext.RequestWrapper, gpp gpplib.GppContainer) (Policy, error) {
	var noSaleBidders []string
	var gppSIDs []int8
	var requestUSPrivacy string
	var warn error

	if req == nil || req.BidRequest == nil {
		return Policy{}, nil
	}

	if req.BidRequest.Regs != nil {
		requestUSPrivacy = req.BidRequest.Regs.USPrivacy
		gppSIDs = req.BidRequest.Regs.GPPSID
	}

	consent, err := SelectCCPAConsent(requestUSPrivacy, gpp, gppSIDs)
	if err != nil {
		warn = &errortypes.Warning{
			Message:     "regs.us_privacy consent does not match uspv1 in GPP, using regs.gpp",
			WarningCode: errortypes.InvalidPrivacyConsentWarningCode}
	}

	if consent == "" && req.Regs != nil {
		consent = req.Regs.USPrivacy
	}
	// Read no sale bidders from request.ext.prebid
	reqExt, err := req.GetRequestExt()
	if err != nil {
		return Policy{}, fmt.Errorf("error reading request.ext: %s", err)
	}
	reqPrebid := reqExt.GetPrebid()
	if reqPrebid != nil {
		noSaleBidders = reqPrebid.NoSale
	}

	return Policy{consent, noSaleBidders}, warn
}

func ReadFromRequest(req *openrtb2.BidRequest) (Policy, error) {
	var gpp gpplib.GppContainer
	if req != nil && req.Regs != nil && len(req.Regs.GPP) > 0 {
		gpp, _ = gpplib.Parse(req.Regs.GPP)
	}

	return ReadFromRequestWrapper(&openrtb_ext.RequestWrapper{BidRequest: req}, gpp)
}

// Write mutates an OpenRTB bid request with the CCPA regulatory information.
func (p Policy) Write(req *openrtb_ext.RequestWrapper) error {
	if req == nil || req.BidRequest == nil {
		return nil
	}

	reqExt, err := req.GetRequestExt()
	if err != nil {
		return err
	}

	if req.Regs == nil {
		req.Regs = &openrtb2.Regs{}
	}
	req.Regs.USPrivacy = p.Consent
	setPrebidNoSale(p.NoSaleBidders, reqExt)
	return nil
}

func SelectCCPAConsent(requestUSPrivacy string, gpp gpplib.GppContainer, gppSIDs []int8) (string, error) {
	var consent string
	var err error

	if len(gpp.SectionTypes) > 0 {
		if gppPolicy.IsSIDInList(gppSIDs, gppConstants.SectionUSPV1) {
			if i := gppPolicy.IndexOfSID(gpp, gppConstants.SectionUSPV1); i >= 0 {
				consent = gpp.Sections[i].GetValue()
			}
		}
	}

	if requestUSPrivacy != "" {
		if consent == "" {
			consent = requestUSPrivacy
		} else if consent != requestUSPrivacy {
			err = errors.New("request.us_privacy consent does not match uspv1")
		}
	}

	return consent, err
}

func setPrebidNoSale(noSaleBidders []string, ext *openrtb_ext.RequestExt) {
	if len(noSaleBidders) == 0 {
		setPrebidNoSaleClear(ext)
	} else {
		setPrebidNoSaleWrite(noSaleBidders, ext)
	}
}

func setPrebidNoSaleClear(ext *openrtb_ext.RequestExt) {
	prebid := ext.GetPrebid()
	if prebid == nil {
		return
	}

	// Remove no sale member
	prebid.NoSale = []string{}
	ext.SetPrebid(prebid)
}

func setPrebidNoSaleWrite(noSaleBidders []string, ext *openrtb_ext.RequestExt) {
	if ext == nil {
		// This should hopefully not be possible. The only caller insures that this has been initialized
		return
	}

	prebid := ext.GetPrebid()
	if prebid == nil {
		prebid = &openrtb_ext.ExtRequestPrebid{}
	}
	prebid.NoSale = noSaleBidders
	ext.SetPrebid(prebid)
}
