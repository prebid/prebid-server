package identity

import (
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache"
)

// FirstPartyKeyExtractor derives the ordered list of alias cache keys for a bid request. Each
// relevant first-party id present becomes one namespaced key; the resolved identity is aliased across
// all of them so a later request carrying any of those ids hits the cache.
//
// Faithful port of Java FirstPartyKeyExtractor.java. Priority (highest first): iiq:<id>
// (intentiq.com), pubcid:<id> (pubcid.org/sharedid.org), maid:<ifa> (device.ifa; upper-cased for CTV
// devicetype 3/7, skipped when device.lmt==1), <source>:<id> for any other eid, then a probabilistic
// dev:<ifa_ua_ip> composite (ua via the deterministic normalizer in useragent.go). De-duplicate
// (first occurrence wins) and cap at maxKeys.
type FirstPartyKeyExtractor struct {
	maxKeys int
}

// sharedSources are the eid sources treated as the shared/pubcid namespace.
var sharedSources = map[string]bool{
	"pubcid.org":   true,
	"sharedid.org": true,
}

// NewFirstPartyKeyExtractor builds the extractor with the per-request key cap.
func NewFirstPartyKeyExtractor(maxKeys int) *FirstPartyKeyExtractor {
	return &FirstPartyKeyExtractor{maxKeys: maxKeys}
}

// CandidateKeys returns the ordered, de-duplicated, capped alias keys for the request.
func (e *FirstPartyKeyExtractor) CandidateKeys(req *openrtb2.BidRequest) []cache.Key {
	if req == nil {
		return nil
	}

	var eids []openrtb2.EID
	if req.User != nil {
		eids = req.User.EIDs
	}
	device := req.Device

	var keys []cache.Key
	keys = addEidKeys(keys, eids, "iiq", cache.ThirdParty, func(s string) bool { return s == iiqSource })
	keys = addEidKeys(keys, eids, "pubcid", cache.FirstParty, func(s string) bool { return sharedSources[s] })
	keys = addMaidKey(keys, device)
	keys = addOtherEidKeys(keys, eids)
	keys = addDeviceComposite(keys, device)

	return e.dedupAndCap(keys)
}

// addEidKeys appends "<namespace>:<id>" keys for every non-blank uid on eids whose source matches.
func addEidKeys(keys []cache.Key, eids []openrtb2.EID, namespace string, keyType cache.KeyType, sourceMatch func(string) bool) []cache.Key {
	for i := range eids {
		eid := eids[i]
		if eid.Source == "" || !sourceMatch(eid.Source) {
			continue
		}
		for _, uid := range eid.UIDs {
			if strings.TrimSpace(uid.ID) == "" {
				continue
			}
			keys = append(keys, cache.Key{Key: namespace + ":" + uid.ID, Type: keyType})
		}
	}
	return keys
}

// addMaidKey appends the "maid:<ifa>" key, skipping a blank ifa or an lmt==1 device and upper-casing
// the ifa for CTV devices (devicetype 3 or 7) to match the resolution request's idtype-8 handling.
func addMaidKey(keys []cache.Key, device *openrtb2.Device) []cache.Key {
	if device == nil || strings.TrimSpace(device.IFA) == "" {
		return keys
	}
	if device.Lmt != nil && *device.Lmt == 1 {
		return keys
	}
	ifa := device.IFA
	if dt := int(device.DeviceType); dt == 3 || dt == 7 {
		ifa = strings.ToUpper(ifa)
	}
	return append(keys, cache.Key{Key: "maid:" + ifa, Type: cache.FirstParty})
}

// addOtherEidKeys appends "<source>:<id>" keys (lower-cased namespace) for every eid source that is
// not the iiq or shared/pubcid source.
func addOtherEidKeys(keys []cache.Key, eids []openrtb2.EID) []cache.Key {
	for i := range eids {
		eid := eids[i]
		source := eid.Source
		if source == "" {
			continue
		}
		if source == iiqSource || sharedSources[source] {
			continue
		}
		namespace := strings.ToLower(source)
		for _, uid := range eid.UIDs {
			if strings.TrimSpace(uid.ID) == "" {
				continue
			}
			keys = append(keys, cache.Key{Key: namespace + ":" + uid.ID, Type: cache.FirstParty})
		}
	}
	return keys
}

// addDeviceComposite appends the probabilistic "dev:<ifa_ua_ip>" composite key, joining the non-empty
// [ifa, normalized-ua, ip||ipv6] fields with "_". The normalized UA (not the raw string) is used to
// avoid fragmenting the cache per request.
func addDeviceComposite(keys []cache.Key, device *openrtb2.Device) []cache.Key {
	if device == nil {
		return keys
	}
	ip := device.IP
	if strings.TrimSpace(ip) == "" {
		ip = device.IPv6
	}
	ua := strings.TrimSpace(normalizeUA(device.UA))

	var fields []string
	for _, f := range []string{device.IFA, ua, ip} {
		if strings.TrimSpace(f) != "" {
			fields = append(fields, f)
		}
	}
	if len(fields) == 0 {
		return keys
	}
	return append(keys, cache.Key{Key: "dev:" + strings.Join(fields, "_"), Type: cache.Device})
}

// dedupAndCap keeps the first occurrence of each key string (preserving order) and caps the result at
// maxKeys.
func (e *FirstPartyKeyExtractor) dedupAndCap(keys []cache.Key) []cache.Key {
	if len(keys) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(keys))
	out := make([]cache.Key, 0, len(keys))
	for _, k := range keys {
		if _, ok := seen[k.Key]; ok {
			continue
		}
		seen[k.Key] = struct{}{}
		out = append(out, k)
		if e.maxKeys > 0 && len(out) >= e.maxKeys {
			break
		}
	}
	return out
}
