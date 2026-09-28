package rtd

import (
	"net/url"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// Segment taxonomy identifiers from the IAB segtax registry:
// https://github.com/InteractiveAdvertisingBureau/openrtb/blob/main/extensions/community_extensions/segtax.md
const (
	segtaxContent10  = 1 // IAB Content Taxonomy 1.0 (deprecated)
	segtaxAudience11 = 4 // IAB Audience Taxonomy 1.1
	segtaxContent22  = 6 // IAB Content Taxonomy 2.2
)

// segtaxExt is the ext object attached to every injected data segment.
type segtaxExt struct {
	Segtax int `json:"segtax"`
}

// resolveDomain picks the most specific domain available on the request. Order
// matters: an explicitly declared domain beats one parsed out of a page URL,
// which in turn beats the publisher's own domain.
func resolveDomain(r *openrtb2.BidRequest) string {
	if r == nil {
		return ""
	}

	var candidates []string
	switch {
	case r.Site != nil:
		candidates = []string{r.Site.Domain, r.Site.Page}
		if r.Site.Publisher != nil {
			candidates = append(candidates, r.Site.Publisher.Domain)
		}
	case r.App != nil:
		candidates = []string{r.App.Domain, r.App.Bundle}
		if r.App.Publisher != nil {
			candidates = append(candidates, r.App.Publisher.Domain)
		}
	case r.DOOH != nil:
		candidates = []string{r.DOOH.Domain}
		if r.DOOH.Publisher != nil {
			candidates = append(candidates, r.DOOH.Publisher.Domain)
		}
	}

	for _, candidate := range candidates {
		if domain := normalizeDomain(candidate); domain != "" {
			return domain
		}
	}
	return ""
}

// normalizeDomain reduces a domain, page URL or app bundle to a bare lowercase
// hostname. It returns "" for values that carry no domain signal.
func normalizeDomain(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}

	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return ""
		}
		value = parsed.Hostname()
	} else {
		// Strip any path and port left on a bare host.
		value, _, _ = strings.Cut(value, "/")
		value, _, _ = strings.Cut(value, ":")
	}

	value = stripHostPrefix(value)
	value = strings.Trim(value, ".")

	if value == "" || !strings.Contains(value, ".") {
		// Rules out "localhost" and iOS store IDs, which the domain classifier
		// cannot interpret.
		return ""
	}
	if isAllDigitsAndDots(value) {
		// Bare IP addresses and numeric app bundles carry no domain semantics.
		return ""
	}
	return value
}

// hostPrefixes are alternate spellings of the same site. Stripping them means
// the desktop, mobile and AMP variants share one cache entry and one
// classification instead of three.
//
// Other subdomains are deliberately preserved: blog., shop. and support. host
// genuinely different content, and collapsing them would mislabel inventory.
var hostPrefixes = []string{"www.", "m.", "amp."}

// stripHostPrefix removes a leading variant prefix, but only when at least two
// labels remain. Without that guard `amp.dev` would reduce to `dev` and be
// discarded as a single-label host.
func stripHostPrefix(host string) string {
	for _, prefix := range hostPrefixes {
		if rest, found := strings.CutPrefix(host, prefix); found && strings.Contains(rest, ".") {
			return rest
		}
	}
	return host
}

func isAllDigitsAndDots(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && r != '.' {
			return false
		}
	}
	return true
}

// enrich writes the resolved segments onto the request. It reports whether
// anything was actually added, so the caller can skip registering a no-op
// mutation.
func (m *Module) enrich(r *openrtb2.BidRequest, s segments) bool {
	if r == nil {
		return false
	}

	changed := false

	// Only materialize a content object if there is something to put in it.
	if len(s.Content22) > 0 || len(s.Content10) > 0 {
		if content := contentOf(r); content != nil {
			if data, ok := m.buildData(content.Data, s.Content22, segtaxContent22); ok {
				content.Data = append(content.Data, data)
				changed = true
			}
			if data, ok := m.buildData(content.Data, s.Content10, segtaxContent10); ok {
				content.Data = append(content.Data, data)
				changed = true
			}
		}
	}

	if len(s.Audience) > 0 {
		var existing []openrtb2.Data
		if r.User != nil {
			existing = r.User.Data
		}
		if data, ok := m.buildData(existing, s.Audience, segtaxAudience11); ok {
			if r.User == nil {
				r.User = &openrtb2.User{}
			}
			r.User.Data = append(r.User.Data, data)
			changed = true
		}
	}

	return changed
}

// buildData assembles one ORTB data object, or reports false when there is
// nothing to add or an equivalent object is already present.
func (m *Module) buildData(existing []openrtb2.Data, ids []string, segtax int) (openrtb2.Data, bool) {
	if len(ids) == 0 || hasDataEntry(existing, m.cfg.DataProviderName, segtax) {
		return openrtb2.Data{}, false
	}

	ext, err := jsonutil.Marshal(segtaxExt{Segtax: segtax})
	if err != nil {
		return openrtb2.Data{}, false
	}

	segs := make([]openrtb2.Segment, 0, len(ids))
	for _, id := range ids {
		segs = append(segs, openrtb2.Segment{ID: id})
	}

	return openrtb2.Data{
		Name:    m.cfg.DataProviderName,
		Segment: segs,
		Ext:     ext,
	}, true
}

// hasDataEntry keeps enrichment idempotent: if this provider already published
// segments for this taxonomy - because an upstream module ran, or because the
// hook is somehow invoked twice - do not append a duplicate.
func hasDataEntry(data []openrtb2.Data, name string, segtax int) bool {
	for _, entry := range data {
		if entry.Name != name || len(entry.Ext) == 0 {
			continue
		}
		var ext segtaxExt
		if err := jsonutil.Unmarshal(entry.Ext, &ext); err != nil {
			continue
		}
		if ext.Segtax == segtax {
			return true
		}
	}
	return false
}

// contentOf returns the content object for whichever distribution channel the
// request declares, creating it when absent. ORTB permits only one of
// site/app/dooh per request.
func contentOf(r *openrtb2.BidRequest) *openrtb2.Content {
	switch {
	case r.Site != nil:
		if r.Site.Content == nil {
			r.Site.Content = &openrtb2.Content{}
		}
		return r.Site.Content
	case r.App != nil:
		if r.App.Content == nil {
			r.App.Content = &openrtb2.Content{}
		}
		return r.App.Content
	case r.DOOH != nil:
		if r.DOOH.Content == nil {
			r.DOOH.Content = &openrtb2.Content{}
		}
		return r.DOOH.Content
	}
	return nil
}

// mutationKey names the field the mutation touches, for the module trace shown
// under ext.prebid.modules.
func mutationKey(r *openrtb2.BidRequest) []string {
	switch {
	case r.Site != nil:
		return []string{"bidrequest", "site", "content", "data"}
	case r.App != nil:
		return []string{"bidrequest", "app", "content", "data"}
	case r.DOOH != nil:
		return []string{"bidrequest", "dooh", "content", "data"}
	}
	return []string{"bidrequest"}
}
