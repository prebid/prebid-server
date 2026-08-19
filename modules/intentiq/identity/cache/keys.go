package cache

import "time"

// KeyType classifies a cache key by the kind of identifier it carries, so the TTL ceiling can be
// applied per id class. Note intentiq.com is ThirdParty, matching how the IntentIQ backend
// classifies the IntentIQ cookie id.
type KeyType int

const (
	FirstParty KeyType = iota
	ThirdParty
	Device
)

// Token is the metric token for a key type (e.g. ThirdParty -> "third_party").
func (t KeyType) Token() string {
	switch t {
	case FirstParty:
		return "first_party"
	case ThirdParty:
		return "third_party"
	case Device:
		return "device"
	default:
		return "unknown"
	}
}

// Key is a namespaced cache key derived from one identifier on the bid request. A request yields an
// ordered list of these; the resolved identity is aliased across all of them.
type Key struct {
	Key  string
	Type KeyType
}

// TTLPolicy governs TTLs for cached entries. The API cttl (or Default when absent) always wins, but
// is capped by a per-KeyType ceiling: we cache the volatile resolved eids, not the stable cookie
// mapping, so the ceilings are deliberately far shorter than the backend's mapping TTLs.
type TTLPolicy struct {
	Default           time.Duration
	FirstPartyCeiling time.Duration
	ThirdPartyCeiling time.Duration
	DeviceCeiling     time.Duration
	NegativeTTL       time.Duration
	InProgressTTL     time.Duration
}

func (p TTLPolicy) CeilingFor(t KeyType) time.Duration {
	switch t {
	case FirstParty:
		return p.FirstPartyCeiling
	case ThirdParty:
		return p.ThirdPartyCeiling
	case Device:
		return p.DeviceCeiling
	default:
		return p.Default
	}
}

func (p TTLPolicy) EffectiveTTL(t KeyType, cttl time.Duration) time.Duration {
	base := p.Default
	if cttl > 0 {
		base = cttl
	}
	return min(base, p.CeilingFor(t))
}

// NegativeTTLFor is the suppression TTL for an unresolvable id. On an empty response the backend
// signals how long to suppress re-querying via cttl; honor it, bounded by the first-party ceiling
// as a guard against absurd values.
func (p TTLPolicy) NegativeTTLFor(cttl time.Duration) time.Duration {
	if cttl > 0 {
		return min(cttl, p.FirstPartyCeiling)
	}
	return p.NegativeTTL
}
