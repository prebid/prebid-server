package tmp

import (
	"github.com/adcontextprotocol/adcp-go/tmproto"
)

// maskGeoMap coarsens a TMP context geo map according to the module's masking
// configuration. The TMP context schema already forbids postcode and lat/long,
// so this operates on the enum-safe fields (metro, region, country, city).
// Returns nil if masking removed everything.
func (m *Module) maskGeoMap(geo map[string]any) map[string]any {
	if geo == nil {
		return nil
	}
	out := make(map[string]any, len(geo))
	for k, v := range geo {
		switch k {
		case "country", "region":
			out[k] = v
		case "metro":
			if m.cfg.Masking.Geo.PreserveMetro {
				out[k] = v
			}
		case "city":
			if m.cfg.Masking.Geo.PreserveCity {
				out[k] = v
			}
		case "zip", "zipcode":
			if m.cfg.Masking.Geo.PreserveZip {
				out[k] = v
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// filterIdentities drops any identity token whose source is not on the
// preserve_eids allowlist. Called for defense-in-depth: mapEIDToUIDType
// already restricts to sources the TMP wire recognizes, but operators may want
// a tighter allowlist per jurisdiction.
func (m *Module) filterIdentities(tokens []tmproto.IdentityToken) []tmproto.IdentityToken {
	if len(m.cfg.Masking.User.PreserveEids) == 0 {
		return tokens
	}
	allowed := make(map[tmproto.UIDType]bool, len(m.cfg.Masking.User.PreserveEids))
	for _, src := range m.cfg.Masking.User.PreserveEids {
		if t := mapEIDToUIDType(src); t != "" {
			allowed[t] = true
		}
	}
	out := make([]tmproto.IdentityToken, 0, len(tokens))
	for _, t := range tokens {
		if allowed[t.UIDType] {
			out = append(out, t)
		}
	}
	return out
}
