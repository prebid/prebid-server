package vast

// Re-export core types from the core package to avoid breaking existing code
// and to provide a convenient import path for module users

import (
	"github.com/prebid/prebid-server/v3/modules/ctv/vast/core"
)

// Re-exported types
type (
	VastResult       = core.VastResult
	SelectedBid      = core.SelectedBid
	CanonicalMeta    = core.CanonicalMeta
	BidSelector      = core.BidSelector
	Enricher         = core.Enricher
	Formatter        = core.Formatter
	ReceiverConfig   = core.ReceiverConfig
	CollisionPolicy  = core.CollisionPolicy
	PlacementRules   = core.PlacementRules
	Placement        = core.Placement
)

// Re-exported constants
const (
	CollisionPolicyError       = core.CollisionPolicyError
	CollisionPolicyVastWins    = core.CollisionPolicyVastWins
	CollisionPolicyOpenRTBWins = core.CollisionPolicyOpenRTBWins
	CollisionPolicyEnrichWins  = core.CollisionPolicyEnrichWins
	CollisionPolicyMerge       = core.CollisionPolicyMerge

	PlacementInline     = core.PlacementInline
	PlacementWrapper    = core.PlacementWrapper
	PlacementExtensions = core.PlacementExtensions
	PlacementSkip       = core.PlacementSkip
	PlacementOmit       = core.PlacementOmit
)
