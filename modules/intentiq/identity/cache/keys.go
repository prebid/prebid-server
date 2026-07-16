package cache

import "time"

// KeyType classifies a cache key by the kind of identifier it carries, so the TTL ceiling can be
// applied per id class (see TTLPolicy). First-party ids are treated as longer-lived than
// third-party / probabilistic ones. Note intentiq.com is treated as ThirdParty, matching how the
// IntentIQ backend classifies the IntentIQ cookie id.
type KeyType int

const (
	FirstParty KeyType = iota
	ThirdParty
	Device
)

// Token is the short, stable, lowercase metric token for a key type
// (e.g. ThirdParty -> "third_party").
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

// Key is a single namespaced cache key derived from a first-party identifier on the bid request,
// together with its KeyType (used to pick the TTL ceiling). A request yields an ordered list of
// these; the resolved identity is aliased across all of them.
type Key struct {
	Key  string
	Type KeyType
}

// TTLPolicy governs TTLs for cached identity entries. The IntentIQ API cttl (or the configured
// default when absent) always wins, but is capped by a per-KeyType ceiling — we cache the volatile
// resolved eids, not the stable cookie mapping, so ceilings are upper bounds only and deliberately
// far shorter than the IntentIQ backend's mapping TTLs. Negative (unresolvable) entries and the
// in-progress marker each use a separate short TTL.
type TTLPolicy struct {
	Default           time.Duration
	FirstPartyCeiling time.Duration
	ThirdPartyCeiling time.Duration
	DeviceCeiling     time.Duration
	NegativeTTL       time.Duration
	InProgressTTL     time.Duration
}

// CeilingFor returns the TTL ceiling for the given key type.
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

// EffectiveTTL is the positive TTL for a key: min(cttl-or-default, ceiling(type)).
func (p TTLPolicy) EffectiveTTL(t KeyType, cttl time.Duration) time.Duration {
	base := p.Default
	if cttl > 0 {
		base = cttl
	}
	return min(base, p.CeilingFor(t))
}

// NegativeTTLFor is the suppression TTL for a negative (unresolvable) entry. On an empty/invalid
// response the IntentIQ backend signals how long to suppress re-querying this user via cttl; honor
// it when present (bounded by the first-party ceiling as a safety cap against absurd values), else
// fall back to the configured default negative TTL.
func (p TTLPolicy) NegativeTTLFor(cttl time.Duration) time.Duration {
	if cttl > 0 {
		return min(cttl, p.FirstPartyCeiling)
	}
	return p.NegativeTTL
}
