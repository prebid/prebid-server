package amp

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
)

// Params defines the parameters of an AMP request.
type Params struct {
	Account         string
	CanonicalURL    string
	Consent         string
	Debug           bool
	Origin          string
	Size            Size
	Slot            string
	StoredRequestID string
	Timeout         *uint64

	ConsentType  int64
	GdprApplies  *bool
	AddtlConsent string
	Targeting    string
}

// Size defines size information of an AMP request.
type Size struct {
	Height         int64
	Multisize      []openrtb2.Format
	OverrideHeight int64
	OverrideWidth  int64
	Width          int64
}

// GDPR consent types
const (
	TCF1 = iota
	TCF2
	CCPA
)

// ParseParams parses the AMP parameters from a HTTP request.
func ParseParams(httpRequest *http.Request) (Params, error) {
	query := httpRequest.URL.Query()

	tagID := query.Get("tag_id")
	if len(tagID) == 0 {
		return Params{}, errors.New("AMP requests require an AMP tag_id")
	}

	// 1) If consent_type is provided, it will be an enum containing the following values:
	// 	    1.1. The contents of gdpr_consent can be treated as TCF V1. We no longer support TCFv1, so ignore any consent_string provided on the request.
	// 	    1.2. The contents of gdpr_consent can be treated as TCF V2. If the consent_string isn't a valid TCF2 string, assume there's no user consent for any purpose as if no gdpr_consent were provided.
	// 	    1.3. The contents of gdpr_consent can be treated as a US Privacy String. If the consent_string isn't a valid USP string, assume for the purposes of opt-out-of-sale enforcement that no gdpr_consent was provided.
	// 	    1.4. Anything else: ignore, log a warning, and assume no gdpr_consent was provided
	// 2) If gdpr_applies is supplied, use its value to set regs.ext.gdpr: gdpr_applies=true means regs.ext.gdpr:1, gdpr_applies=false means regs.ext.gdpr:0, any other value means regs.ext.gdpr is not set.
	// 3) If consent_type="2", and gdpr_consent is not empty, then copy gdpr_consent to user.ext.consent
	// 4) If consent_type="3", and gdpr_consent is not empty, then copy gdpr_consent to regs.ext.us_privacy
	// 5) If addtl_consent is supplied, copy its value to user.ext.ConsentedProvidersSettings.consented_providers
	params := Params{
		Account:      query.Get("account"),
		CanonicalURL: query.Get("curl"),
		Consent:      chooseConsent(query.Get("consent_string"), query.Get("gdpr_consent")),
		Debug:        query.Get("debug") == "1",
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
		Timeout:         parseIntPtr(query.Get("timeout")),
		ConsentType:     parseInt(query.Get("consent_type")),
		GdprApplies:     parseBoolPtr(query.Get("gdpr_applies")),
		AddtlConsent:    query.Get("addtl_consent"),
		Targeting:       query.Get("targeting"),
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
	var rv bool = false
	switch value {
	case "true":
		rv = true
		return &rv
	case "false":
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
