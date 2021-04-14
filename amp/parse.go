package amp

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
)

// Params defines the paramters of an AMP request.
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
}

// Size defines size information of an AMP request.
type Size struct {
	Height         int64
	Multisize      []openrtb2.Format
	OverrideHeight int64
	OverrideWidth  int64
	Width          int64
}

// ParseParams parses the AMP paramters from a HTTP request.
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

	// Fallback to 'gdpr_consent' for compatability until it's no longer used. This was our original
	// implementation before the same AMP macro was reused for CCPA.
	return gdprConsent
}
