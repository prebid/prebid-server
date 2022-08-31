package amp

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
)

// Params defines the parameters of an AMP request.
type Params struct {
	Account         string
	CanonicalURL    string
	Consent         string
	ConsentType     int64
	Debug           bool
	GdprApplies     *bool
	Origin          string
	Size            Size
	Slot            string
	StoredRequestID string
	Targeting       string
	Timeout         *uint64
}

// Size defines size information of an AMP request.
type Size struct {
	Height         int64
	Multisize      []openrtb2.Format
	OverrideHeight int64
	OverrideWidth  int64
	Width          int64
}

// Policy consent types
const (
	ConsentNone      = 0
	ConsentTCF1      = 1
	ConsentTCF2      = 2
	ConsentUSPrivacy = 3
)

func ReadPolicy(ampParams Params, req *openrtb2.BidRequest, pbsConfigGDPREnabled bool) (privacy.PolicyWriter, error) {
	if len(ampParams.Consent) == 0 {
		return privacy.NilPolicyWriter{}, nil
	}

	var rv privacy.PolicyWriter = privacy.NilPolicyWriter{}
	var warning error
	var errMsg string

	// If consent_type was set to CCPA or GDPR TCF2, we return a valid writer even if the consent string is invalid
	switch ampParams.ConsentType {
	case ConsentTCF2:
		if pbsConfigGDPREnabled {
			rv = buildGdprTCF2ConsentWriter(ampParams)
			// Log warning if GDPR consent string is invalid
			errMsg = validateTCf2ConsentString(ampParams.Consent)
		}
	case ConsentUSPrivacy:
		rv = ccpa.ConsentWriter{ampParams.Consent}
		if ccpa.ValidateConsent(ampParams.Consent) {
			if parseGdprApplies(ampParams.GdprApplies) == 1 {
				// Log warning because AMP request comes with both a valid CCPA string and gdpr_applies set to true
				errMsg = "AMP request gdpr_applies value was ignored in account of provided consent string found to be CCPA and not GDPR."
			}
		} else {
			// Log warning if CCPA string is invalid
			errMsg = fmt.Sprintf("Consent string '%s' is not a valid CCPA consent string.", ampParams.Consent)
		}
	case ConsentTCF1:
		errMsg = "TCF1 consent is deprecated and no longer supported."
	case ConsentNone:
		fallthrough
	default:
		if ccpa.ValidateConsent(ampParams.Consent) {
			rv = ccpa.ConsentWriter{ampParams.Consent}
			if parseGdprApplies(ampParams.GdprApplies) == 1 {
				errMsg = "AMP request gdpr_applies value was ignored in account of provided consent string found to be CCPA and not GDPR."
			}
		} else if pbsConfigGDPREnabled && len(validateTCf2ConsentString(ampParams.Consent)) == 0 {
			rv = buildGdprTCF2ConsentWriter(ampParams)
		} else {
			errMsg = fmt.Sprintf("Consent '%s' is not recognized as CCPA nor GDPR TCF2.", ampParams.Consent)
		}
	}

	if len(errMsg) > 0 {
		warning = &errortypes.Warning{
			Message:     errMsg,
			WarningCode: errortypes.InvalidPrivacyConsentWarningCode,
		}
	}
	return rv, warning
}

// buildGdprTCF2ConsentWriter returns a gdpr.ConsentWriter that will set regs.ext.gdpr to the value of 1 if gdpr_applies wasn't defined. If gdpr_applies
// was defined, sets regs.ext.gdpr to either 0 or 1
func buildGdprTCF2ConsentWriter(ampParams Params) gdpr.ConsentWriter {
	writer := gdpr.ConsentWriter{Consent: ampParams.Consent}

	// set regs.ext.gdpr:1 if gdpr_applies is not set, else regs.ext.gdpr should hold gdpr_applies value
	var gdprValue int8 = 1
	if ampParams.GdprApplies == nil {
		gdprValue = 1
	} else {
		gdprValue = parseGdprApplies(ampParams.GdprApplies)
	}
	writer.RegExtGDPR = &gdprValue

	return writer
}

// parseGdprApplies returns a 0 if gdprApplies was not set or if false, and a 1 if gdprApplies was set to true
func parseGdprApplies(gdprApplies *bool) int8 {
	gdpr := int8(0)
	if gdprApplies == nil {
		return gdpr
	}

	if *gdprApplies {
		gdpr = int8(1)
	}

	return gdpr
}

// ParseParams parses the AMP parameters from a HTTP request.
func validateTCf2ConsentString(consent string) string {
	if tcf2.IsConsentV2(consent) {
		if _, err := tcf2.ParseString(consent); err != nil {
			return err.Error()
		}
	} else {
		return fmt.Sprintf("Consent string '%s' is not a valid TCF2 consent string.", consent)
	}
	return ""
}

// ParseParams parses the AMP parameters from a HTTP request.
func ParseParams(httpRequest *http.Request) (Params, error) {
	query := httpRequest.URL.Query()

	tagID := query.Get("tag_id")
	if len(tagID) == 0 {
		return Params{}, errors.New("AMP requests require an AMP tag_id")
	}

	params := Params{
		Account:      query.Get("account"),
		CanonicalURL: query.Get("curl"),
		Consent:      chooseConsent(query.Get("consent_string"), query.Get("gdpr_consent")),
		ConsentType:  parseInt(query.Get("consent_type")),
		Debug:        query.Get("debug") == "1",
		GdprApplies:  parseBoolPtr(query.Get("gdpr_applies")),
		Origin:       query.Get("__amp_source_origin"),
		Size: Size{
			Height:         parseInt(query.Get("h")),
			Multisize:      parseMultisize(query.Get("ms")),
			OverrideHeight: parseInt(query.Get("oh")),
			OverrideWidth:  parseInt(query.Get("ow")),
			Width:          parseInt(query.Get("w")),
		},
		Slot:            query.Get("slot"),
		StoredRequestID: tagID,
		Targeting:       query.Get("targeting"),
		Timeout:         parseIntPtr(query.Get("timeout")),
	}
	return params, nil
}

func parseIntPtr(value string) *uint64 {
	if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
		return &parsed
	}
	return nil
}

func parseInt(value string) int64 {
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return parsed
	}
	return 0
}

func parseBoolPtr(value string) *bool {
	if rv, err := strconv.ParseBool(value); err == nil {
		return &rv
	}
	return nil
}

func parseMultisize(multisize string) []openrtb2.Format {
	if multisize == "" {
		return nil
	}

	sizeStrings := strings.Split(multisize, ",")
	sizes := make([]openrtb2.Format, 0, len(sizeStrings))
	for _, sizeString := range sizeStrings {
		wh := strings.Split(sizeString, "x")
		if len(wh) != 2 {
			return nil
		}
		f := openrtb2.Format{
			W: parseInt(wh[0]),
			H: parseInt(wh[1]),
		}
		if f.W == 0 && f.H == 0 {
			return nil
		}

		sizes = append(sizes, f)
	}
	return sizes
}

func chooseConsent(consent, gdprConsent string) string {
	if len(consent) > 0 {
		return consent
	}

	// Fallback to 'gdpr_consent' for compatibility until it's no longer used. This was our original
	// implementation before the same AMP macro was reused for CCPA.
	return gdprConsent
}
