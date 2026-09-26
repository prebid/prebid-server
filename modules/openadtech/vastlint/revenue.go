package vastlint

// revenueRules are the vastlint catalog ids whose violation is a direct
// revenue or measurement loss. The Go binding does not expose the flag, so
// this set stays aligned with vastlint-core RuleMeta.revenue_impact.
var revenueRules = map[string]struct{}{
	"VAST-2.0-mediafile-https":              {},
	"VAST-2.0-tracking-https":               {},
	"VAST-2.0-duplicate-impression":         {},
	"VAST-4.1-mezzanine-recommended":        {},
	"VAST-4.1-vpaid-in-interactive-context": {},
	"VAST-2.0-linear-tracking-quartiles":    {},
	"VAST-2.0-inline-impression":            {},
	"VAST-2.0-wrapper-impression":           {},
	"VAST-2.0-wrapper-vastadtaguri":         {},
	"VAST-2.0-url-empty":                    {},
	"VAST-4.1-vpaid-apiframework":           {},
	"VAST-2.0-flash-mediafile":              {},
}

func revenueImpact(ruleID string) bool {
	_, ok := revenueRules[ruleID]
	return ok
}
