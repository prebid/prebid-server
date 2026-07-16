package cache

import "github.com/prebid/openrtb/v20/openrtb2"

// State is the outcome of a multi-key cache lookup.
type State int

const (
	// Miss — nothing cached; fetch from the API.
	Miss State = iota
	// Hit — a positive entry was found; Eids carries the resolved identity.
	Hit
	// Negative — a negative sentinel was found (the id is known-unresolvable); skip the upstream
	// call and do not enrich.
	Negative
	// InProgress — a resolution call for this id is already in flight; skip the upstream call (do
	// not fire a duplicate) and do not enrich.
	InProgress
)

// Layer identifies which cache layer served the outcome: L1 (in-process) or L2 (Redis).
type Layer int

const (
	// LayerNone is used for a full miss, where no layer matched.
	LayerNone Layer = iota
	// LayerL1 is the in-process layer (freecache).
	LayerL1
	// LayerL2 is the shared layer (Redis by default).
	LayerL2
)

// Token is the short, stable, lowercase metric token for a layer ("l1"/"l2").
func (l Layer) Token() string {
	switch l {
	case LayerL1:
		return "l1"
	case LayerL2:
		return "l2"
	default:
		return "unknown"
	}
}

// Result is the outcome of a multi-key cache lookup. KeyType is the type of the candidate key that
// produced the outcome (Hit/Negative/InProgress); both KeyType and Layer are zero-valued for Miss,
// where no key/layer matched.
type Result struct {
	State   State
	Eids    []openrtb2.EID
	KeyType KeyType
	Layer   Layer
}

// HitResult builds a Hit result.
func HitResult(eids []openrtb2.EID, keyType KeyType, layer Layer) Result {
	return Result{State: Hit, Eids: eids, KeyType: keyType, Layer: layer}
}

// NegativeResult builds a Negative result.
func NegativeResult(keyType KeyType, layer Layer) Result {
	return Result{State: Negative, KeyType: keyType, Layer: layer}
}

// InProgressResult builds an InProgress result.
func InProgressResult(keyType KeyType, layer Layer) Result {
	return Result{State: InProgress, KeyType: keyType, Layer: layer}
}

// MissResult builds a Miss result.
func MissResult() Result {
	return Result{State: Miss}
}
