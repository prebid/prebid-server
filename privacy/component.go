package privacy

import (
	"strings"
)

const (
	ComponentTypeBidder       = "bidder"
	ComponentTypeAnalytics    = "analytics"
	ComponentTypeRealTimeData = "rtd"
	ComponentTypeGeneral      = "general"
)

type Component struct {
	Type string
	Name string
}

func (c Component) MatchesName(v string) bool {
	return strings.EqualFold(c.Name, v)
}

func (c Component) MatchesType(v string) bool {
	return strings.EqualFold(c.Type, v)
}
