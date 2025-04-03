package optimizationmodule

import (
	"encoding/json"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// will be used in Build rules trie
func NewSchemaFunctionFactory(name string, params []string) Function {
	switch name {
	case "deviceCountry":
		return NewDeviceCountry(params)
	case "dataCenters":
		return NewDatacenters(params)
	case "channel":
		return NewChannel()
	default:
		return nil
	}
}

func NewResultFunctionFactory(name string, params json.RawMessage) Function {
	switch name {
	case "setDeviceIP":
		return NewSetDevIp(params)
	case "excludeBidders":
		return NewExcludeBidders(params)
	default:
		return nil
	}
}

func BuildRulesTree(data json.RawMessage) *Rules {
	var conf Conf

	if err := jsonutil.Unmarshal(data, &conf); err != nil {
		return nil
	}

	currNode := &Node{}
	rules := Rules{Root: currNode}

	for _, rule := range conf.Rule {
		for ci, condition := range rule.Conditions {

			if len(currNode.Children) == 0 {
				currNode.Children = make(map[string]*Node, 0)
				f := NewSchemaFunctionFactory(conf.Schema[ci].Func, conf.Schema[ci].Args)
				currNode.Function = f
			}

			_, ok := currNode.Children[condition]
			if ok {
				currNode = currNode.Children[condition]
			} else {
				nextNode := &Node{}
				currNode.Children[condition] = nextNode
				currNode = nextNode
			}
		}

		// array of func
		f := NewResultFunctionFactory(rule.Results[0].Func, rule.Results[0].Args)
		currNode.Function = f
		currNode = rules.Root

	}

	return &rules
}
