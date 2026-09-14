package cache

import "github.com/prebid/openrtb/v20/openrtb2"

// State is the outcome of a multi-key lookup. Negative and InProgress both mean: skip the upstream
// call and do not enrich.
type State int

const (
	Miss State = iota
	Hit
	Negative
	InProgress
)

type Layer int

const (
	// LayerNone is the zero value, used for a full miss.
	LayerNone Layer = iota
	LayerL1
	LayerL2
)

// Token is the metric token for a layer: "l1"/"l2" when an entry was found, "none" for a full miss
func (l Layer) Token() string {
	switch l {
	case LayerL1:
		return "l1"
	case LayerL2:
		return "l2"
	default:
		return "none"
	}
}

// Result carries the outcome plus which candidate key and layer produced it. KeyType and Layer are
// zero-valued for a Miss. AbTestUUID/Tc ride along on Hit and Negative so the impression report can
// attribute a cached outcome; an in-progress marker has no resolution outcome, so it carries neither.
type Result struct {
	State      State
	Eids       []openrtb2.EID
	AbTestUUID string
	Tc         *int64
	KeyType    KeyType
	Layer      Layer
}

func HitResult(eids []openrtb2.EID, abTestUUID string, tc *int64, keyType KeyType, layer Layer) Result {
	return Result{State: Hit, Eids: eids, AbTestUUID: abTestUUID, Tc: tc, KeyType: keyType, Layer: layer}
}

func NegativeResult(abTestUUID string, tc *int64, keyType KeyType, layer Layer) Result {
	return Result{State: Negative, AbTestUUID: abTestUUID, Tc: tc, KeyType: keyType, Layer: layer}
}

func InProgressResult(keyType KeyType, layer Layer) Result {
	return Result{State: InProgress, KeyType: keyType, Layer: layer}
}

func MissResult() Result {
	return Result{State: Miss}
}

func toResult(entry Entry, keyType KeyType, layer Layer) Result {
	if entry.InProgress {
		return InProgressResult(keyType, layer)
	}
	if entry.Negative {
		return NegativeResult(entry.AbTestUUID, entry.Tc, keyType, layer)
	}
	return HitResult(entry.Eids, entry.AbTestUUID, entry.Tc, keyType, layer)
}
