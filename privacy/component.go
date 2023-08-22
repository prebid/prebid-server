package privacy

import (
	"errors"
	"fmt"
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

func (c Component) Matches(target Component) bool {
	return strings.EqualFold(c.Type, target.Type) && (c.Name == "*" || strings.EqualFold(c.Name, target.Name))
}

var ErrComponentEmpty = errors.New("unable to parse empty component")

func ParseComponent(v string) (Component, error) {
	if len(v) == 0 {
		return Component{}, ErrComponentEmpty
	}

	split := strings.Split(v, ".")

	if len(split) == 2 {
		if !validComponentType(split[0]) {
			return Component{}, fmt.Errorf("unable to parse component (invalid type): %s", v)
		}
		return Component{
			Type: split[0],
			Name: split[1],
		}, nil
	}

	if len(split) == 1 {
		return Component{
			Name: split[0],
		}, nil
	}

	return Component{}, fmt.Errorf("unable to parse component: %s", v)
}

func validComponentType(t string) bool {
	t = strings.ToLower(t)

	if t == ComponentTypeBidder ||
		t == ComponentTypeAnalytics ||
		t == ComponentTypeRealTimeData ||
		t == ComponentTypeGeneral {
		return true
	}

	return false
}
