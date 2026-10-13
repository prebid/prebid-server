package vastlint

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// The initial raw-creative policy supports explicitly non-GDPR traffic with no
// other supplied consent or restriction signals. It does not interpret consent
// strings or claim that activity permission constitutes consent. Unknown,
// contradictory, malformed and oversized signals suppress export.
func creativePrivacyAllowed(rw *openrtb_ext.RequestWrapper) bool {
	if rw == nil || rw.BidRequest == nil || rw.Regs == nil || rw.GDPRSignalConflict {
		return false
	}
	regs := rw.Regs
	if regs.COPPA != 0 || regs.USPrivacy != "" || regs.GPP != "" || len(regs.GPPSID) != 0 {
		return false
	}
	if len(regs.Ext) > maxExtensionBytes {
		return false
	}
	// Header-derived GPC may still live in the wrapper's parsed extension.
	// Reading the serialized extension alone can miss that restriction.
	regExt, err := rw.GetRegExt()
	if err != nil || regExt.GetGPC() != nil {
		return false
	}
	gdpr := regs.GDPR
	if len(regs.Ext) != 0 {
		var ext struct {
			GPC       json.RawMessage `json:"gpc"`
			GDPR      *int8           `json:"gdpr"`
			USPrivacy string          `json:"us_privacy"`
			GPP       string          `json:"gpp"`
			GPPSID    []int8          `json:"gpp_sid"`
		}
		if json.Unmarshal(regs.Ext, &ext) != nil {
			return false
		}
		if len(ext.GPC) != 0 || ext.USPrivacy != "" || ext.GPP != "" || len(ext.GPPSID) != 0 || (ext.GDPR != nil && *ext.GDPR != 0) {
			return false
		}
		if gdpr == nil {
			gdpr = ext.GDPR
		}
	}
	if gdpr == nil || *gdpr != 0 {
		return false
	}
	if device := rw.Device; device != nil {
		if device.DNT != nil && *device.DNT != 0 || device.Lmt != nil && *device.Lmt != 0 {
			return false
		}
	}
	if user := rw.User; user != nil {
		if user.Consent != "" || len(user.Ext) > maxExtensionBytes {
			return false
		}
		if len(user.Ext) != 0 {
			var ext struct {
				Consent        string          `json:"consent"`
				Additional     json.RawMessage `json:"ConsentedProvidersSettings"`
				AdditionalList json.RawMessage `json:"consented_providers_settings"`
			}
			if json.Unmarshal(user.Ext, &ext) != nil || ext.Consent != "" || len(ext.Additional) != 0 || len(ext.AdditionalList) != 0 {
				return false
			}
		}
	}
	return true
}
