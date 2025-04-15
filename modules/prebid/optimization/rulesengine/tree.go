package rulesengine

import (
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	structs "github.com/prebid/prebid-server/v3/modules/prebid/optimization/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type Tree struct {
	Root *Node
}

type Node struct {
	SchemaFunction  SchemaFunction
	ResultFunctions []ResultFunction
	Children        map[string]*Node
}

func Execute(r *Tree, rw *openrtb_ext.RequestWrapper) (*hookstage.ChangeSet[hookstage.BidderRequestPayload], error) {
	currNode := r.Root

	for len(currNode.Children) > 0 {
		// schema function
		res, err := currNode.SchemaFunction.Call(rw)
		if err != nil {
			return nil, err
		}

		next := currNode.Children[res]

		if next != nil {
			currNode = next
		}
	}

	changeSet := &hookstage.ChangeSet[hookstage.BidderRequestPayload]{}
	for _, rf := range currNode.ResultFunctions {
		err := rf.AddChangeSet(changeSet)
		if err != nil {
			return nil, err
		}
	}

	return changeSet, nil
}

func BuildRulesTree(conf structs.ModelGroup) (*Tree, error) {

	currNode := &Node{}
	rules := Tree{Root: currNode}

	for _, rule := range conf.Rule {
		for ci, condition := range rule.Conditions {

			if len(currNode.Children) == 0 {
				currNode.Children = make(map[string]*Node, 0)
				f, err := NewSchemaFunctionFactory(conf.Schema[ci].Func, conf.Schema[ci].Args)
				if err != nil {
					return nil, err
				}
				currNode.SchemaFunction = f
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
		for _, res := range rule.Results {
			resFunc, err := NewResultFunctionFactory(res.Func, res.Args)
			if err != nil {
				return nil, err
			}
			currNode.ResultFunctions = append(currNode.ResultFunctions, resFunc)
		}

		currNode = rules.Root

	}

	return &rules, nil
}
