package trafficshaping

import (
	"encoding/json"
	"hash/fnv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// shouldSkipByRate determines if shaping should be skipped based on skipRate
func shouldSkipByRate(requestID string, skipRate int, salt string) bool {
	if skipRate <= 0 {
		return false
	}
	if skipRate >= 100 {
		return true
	}

	// Compute deterministic sample: fnv1a32(salt + requestID) % 100
	h := fnv.New32a()
	h.Write([]byte(salt))
	h.Write([]byte(requestID))
	sample := int(h.Sum32() % 100)

	return sample < skipRate
}

// shouldSkipByCountry determines if shaping should be skipped based on country
func shouldSkipByCountry(wrapper *openrtb_ext.RequestWrapper, allowedCountries map[string]struct{}) bool {
	// If no country restriction, don't skip
	if len(allowedCountries) == 0 {
		return false
	}

	// Get country from device.geo
	if wrapper == nil || wrapper.BidRequest == nil || wrapper.Device == nil || wrapper.Device.Geo == nil {
		return true // Skip if no geo data
	}

	country := wrapper.Device.Geo.Country
	if country == "" {
		return true // Skip if country is empty
	}

	_, allowed := allowedCountries[country]
	return !allowed
}

// getGPID extracts the GPID from an impression
func getGPID(impWrapper *openrtb_ext.ImpWrapper) string {
	if impWrapper == nil || impWrapper.Imp == nil {
		return ""
	}

	// Try imp.ext.gpid first
	impExt, err := impWrapper.GetImpExt()
	if err == nil && impExt != nil {
		ext := impExt.GetExt()
		if gpidRaw, ok := ext["gpid"]; ok {
			var gpid string
			if err := jsonutil.Unmarshal(gpidRaw, &gpid); err == nil && gpid != "" {
				return gpid
			}
		}
	}

	// Fallback to imp.ext.data.adserver.adslot
	if impExt != nil {
		ext := impExt.GetExt()
		if dataRaw, ok := ext["data"]; ok {
			var data struct {
				AdServer struct {
					AdSlot string `json:"adslot"`
				} `json:"adserver"`
			}
			if err := jsonutil.Unmarshal(dataRaw, &data); err == nil && data.AdServer.AdSlot != "" {
				return data.AdServer.AdSlot
			}
		}
	}

	return ""
}

// getAllowedBidders returns the set of allowed bidders for a GPID
func getAllowedBidders(gpid string, config *ShapingConfig) map[string]struct{} {
	if config == nil {
		return nil
	}

	rule, ok := config.GPIDRules[gpid]
	if !ok {
		return nil
	}

	return rule.AllowedBidders
}

// filterBannerSizes filters banner sizes to only allowed sizes
func filterBannerSizes(imp *openrtb2.Imp, allowedSizes map[BannerSize]struct{}) {
	if imp == nil || imp.Banner == nil || len(allowedSizes) == 0 {
		return
	}

	// If banner has formats, filter them
	if len(imp.Banner.Format) > 0 {
		filtered := make([]openrtb2.Format, 0, len(imp.Banner.Format))
		for _, format := range imp.Banner.Format {
			size := BannerSize{W: format.W, H: format.H}
			if _, allowed := allowedSizes[size]; allowed {
				filtered = append(filtered, format)
			}
		}

		// Only update if we have at least one allowed size
		if len(filtered) > 0 {
			imp.Banner.Format = filtered
		}
		return
	}

	// If banner has w/h, check if allowed
	if imp.Banner.W != nil && imp.Banner.H != nil {
		size := BannerSize{W: *imp.Banner.W, H: *imp.Banner.H}
		if _, allowed := allowedSizes[size]; !allowed {
			// Convert to format if not allowed
			// But only if we have allowed sizes to use
			if len(allowedSizes) > 0 {
				formats := make([]openrtb2.Format, 0, len(allowedSizes))
				for allowedSize := range allowedSizes {
					formats = append(formats, openrtb2.Format{
						W: allowedSize.W,
						H: allowedSize.H,
					})
				}
				imp.Banner.Format = formats
				imp.Banner.W = nil
				imp.Banner.H = nil
			}
		}
	}
}

// pruneEIDs filters user.ext.eids to only allowed vendors
func pruneEIDs(wrapper *openrtb_ext.RequestWrapper, allowedVendors map[string]struct{}) error {
	if len(allowedVendors) == 0 {
		return nil
	}

	if wrapper == nil || wrapper.BidRequest == nil || wrapper.User == nil {
		return nil
	}

	userExt, err := wrapper.GetUserExt()
	if err != nil {
		return err
	}

	eids := userExt.GetEid()
	if eids == nil || len(*eids) == 0 {
		return nil
	}

	filtered := make([]openrtb2.EID, 0, len(*eids))
	for _, eid := range *eids {
		if shouldKeepEID(eid, allowedVendors) {
			filtered = append(filtered, eid)
		}
	}

	userExt.SetEid(&filtered)
	return nil
}

// vendorPatternMappings maps shorthand vendor identifiers to substrings expected in EID sources.
var vendorPatternMappings = map[string][]string{
	"33acrossId": {"33across.com"},
	"criteoId":   {"criteo.com"},
	"hadronId":   {"audigent.com", "hadron"},
	"idl_env":    {"liveramp.com", "identitylink"},
	"index":      {"casalemedia.com"},
	"magnite":    {"rubiconproject.com"},
	"medianet":   {"media.net"},
	"openx":      {"openx.net"},
	"pubcid":     {"pubcid.org"},
	"pubmatic":   {"pubmatic.com"},
	"sovrn":      {"liveintent.com", "sovrn"},
	"tdid":       {"adserver.org"},
	"uid2":       {"uidapi.com"},
}

// shouldKeepEID determines if an EID should be kept based on allowed vendors
func shouldKeepEID(eid openrtb2.EID, allowedVendors map[string]struct{}) bool {
	source := strings.ToLower(eid.Source)

	// Direct match
	if _, ok := allowedVendors[source]; ok {
		return true
	}

	// Vendor-specific mappings (conservative, substring match)
	for vendor := range allowedVendors {
		if patterns, ok := vendorPatternMappings[vendor]; ok {
			for _, pattern := range patterns {
				if strings.Contains(source, pattern) {
					// Special case for TDID: check rtiPartner
					if vendor == "tdid" && source == "adserver.org" {
						return checkTDIDRtiPartner(eid)
					}
					return true
				}
			}
		}
	}

	// If ambiguous, keep it (fail-open)
	return true
}

// checkTDIDRtiPartner checks if the EID has the correct rtiPartner for TDID
func checkTDIDRtiPartner(eid openrtb2.EID) bool {
	// For TDID, we expect source=adserver.org and check UIDs for rtiPartner=TDID
	for _, uid := range eid.UIDs {
		if uid.Ext != nil {
			var ext struct {
				RtiPartner string `json:"rtiPartner"`
			}
			if err := json.Unmarshal(uid.Ext, &ext); err == nil {
				if strings.ToUpper(ext.RtiPartner) == "TDID" {
					return true
				}
			}
		}
	}
	return false
}
