package amp

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/privacy/ccpa"
	"github.com/prebid/prebid-server/v3/privacy/gdpr"
)

// Params defines the parameters of an AMP request.
type Params struct {
	Account           string
	AdditionalConsent string
	CanonicalURL      string
	Consent           string
	ConsentType       int64
	Debug             bool
	GdprApplies       *bool
	Origin            string
	Size              Size
	Slot              string
	StoredRequestID   string
	Targeting         string
	Timeout           *uint64
	Trace             string
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

// ReadPolicy returns a privacy writer in accordance to the query values consent, consent_type and gdpr_applies.
// Returned policy writer could either be GDPR, CCPA or NilPolicy. The second return value is a warning.
func ReadPolicy(ampParams Params, pbsConfigGDPREnabled bool) (privacy.PolicyWriter, error) {
	if len(ampParams.Consent) == 0 {
		return privacy.NilPolicyWriter{}, nil
	}

	var rv privacy.PolicyWriter = privacy.NilPolicyWriter{}
	var warning error
	var warningMsg string

	// If consent_type was set to CCPA or GDPR TCF2, we return a valid writer even if the consent string is invalid
	switch ampParams.ConsentType {
	case ConsentTCF1:
		warningMsg = "TCF1 consent is deprecated and no longer supported."
	case ConsentTCF2:
		if pbsConfigGDPREnabled {
			rv = buildGdprTCF2ConsentWriter(ampParams)
			// Log warning if GDPR consent string is invalid
			warningMsg = validateTCf2ConsentString(ampParams.Consent)
		}
	case ConsentUSPrivacy:
		rv = ccpa.ConsentWriter{Consent: ampParams.Consent}
		if ccpa.ValidateConsent(ampParams.Consent) {
			if parseGdprApplies(ampParams.GdprApplies) == 1 {
				// Log warning because AMP request comes with both a valid CCPA string and gdpr_applies set to true
				warningMsg = "AMP request gdpr_applies value was ignored because provided consent string is a CCPA consent string"
			}
		} else {
			// Log warning if CCPA string is invalid
			warningMsg = fmt.Sprintf("Consent string '%s' is not a valid CCPA consent string.", ampParams.Consent)
		}
	default:
		if ccpa.ValidateConsent(ampParams.Consent) {
			rv = ccpa.ConsentWriter{Consent: ampParams.Consent}
			if parseGdprApplies(ampParams.GdprApplies) == 1 {
				warningMsg = "AMP request gdpr_applies value was ignored because provided consent string is a CCPA consent string"
			}
		} else if pbsConfigGDPREnabled && len(validateTCf2ConsentString(ampParams.Consent)) == 0 {
			rv = buildGdprTCF2ConsentWriter(ampParams)
		} else {
			warningMsg = fmt.Sprintf("Consent string '%s' is not recognized as one of the supported formats CCPA or TCF2.", ampParams.Consent)
		}
	}

	if len(warningMsg) > 0 {
		warning = &errortypes.Warning{
			Message:     warningMsg,
			WarningCode: errortypes.InvalidPrivacyConsentWarningCode,
		}
	}
	return rv, warning
}

// buildGdprTCF2ConsentWriter returns a gdpr.ConsentWriter that will set regs.ext.gdpr to the value
// of 1 if gdpr_applies wasn't defined. The reason for this is that this function gets called when
// GDPR applies, even if field gdpr_applies wasn't set in the AMP endpoint query.
func buildGdprTCF2ConsentWriter(ampParams Params) gdpr.ConsentWriter {
	writer := gdpr.ConsentWriter{Consent: ampParams.Consent}

	// If gdpr_applies was not set, regs.ext.gdpr must equal 1
	var gdprValue int8 = 1
	if ampParams.GdprApplies != nil {
		// set regs.ext.gdpr if non-nil gdpr_applies was set to true
		gdprValue = parseGdprApplies(ampParams.GdprApplies)
	}
	writer.GDPR = &gdprValue

	return writer
}

// parseGdprApplies returns a 0 if gdprApplies was not set or if false, and a 1 if
// gdprApplies was set to true
func parseGdprApplies(gdprApplies *bool) int8 {
	gdpr := int8(0)

	if gdprApplies != nil && *gdprApplies {
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
		Account:           query.Get("account"),
		AdditionalConsent: query.Get("addtl_consent"),
		CanonicalURL:      query.Get("curl"),
		Consent:           chooseConsent(query.Get("consent_string"), query.Get("gdpr_consent")),
		ConsentType:       parseInt(query.Get("consent_type")),
		Debug:             query.Get("debug") == "1",
		Origin:            query.Get("__amp_source_origin"),
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
		Trace:           query.Get("trace"),
	}
	var err error
	urlQueryGdprApplies := query.Get("gdpr_applies")
	if len(urlQueryGdprApplies) > 0 {
		if params.GdprApplies, err = parseBoolPtr(urlQueryGdprApplies); err != nil {
			return params, err
		}
	}

	urlQueryTimeout := query.Get("timeout")
	if len(urlQueryTimeout) > 0 {
		if params.Timeout, err = parseIntPtr(urlQueryTimeout); err != nil {
			return params, err
		}
	}

	return params, nil
}

func parseIntPtr(value string) (*uint64, error) {
	var rv uint64
	var err error

	if rv, err = strconv.ParseUint(value, 10, 64); err != nil {
		return nil, err
	}
	return &rv, nil
}

func parseInt(value string) int64 {
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return parsed
	}
	return 0
}

func parseBoolPtr(value string) (*bool, error) {
	var rv bool
	var err error

	if rv, err = strconv.ParseBool(value); err != nil {
		return nil, err
	}
	return &rv, nil
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
