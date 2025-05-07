package rulesengine

import (
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v3/rules"
)

// treeBuilder is a custom tree builder for the rules engine module.
// It implements the TreeBuilder interface and is used to build a tree structure
// based on the provided configuration.
type treeBuilder[T1 any, T2 any] struct {
	Config            config.ModelGroup
	SchemaFuncFactory rules.SchemaFuncFactory[T1]
	ResultFuncFactory rules.ResultFuncFactory[T2]
}

// Build constructs the tree based on the provided configuration.
// It iterates through the rules and conditions, creating nodes and setting
// schema and result functions as needed. The tree is built in a way that
// allows for efficient traversal and execution of the rules.
// The function returns an error if there is an issue with the configuration
// or if the tree cannot be built successfully.
// The function is generic and can work with any types T1 and T2.
// It is expected that T1 and T2 are the types of the request and result payloads respectively.
// The function uses the provided schema and result function factories to create
// the appropriate functions for each node in the tree.
func (tb *treeBuilder[T1, T2]) Build(tree *rules.Tree[T1, T2]) error {
	currNode := tree.Root

	for _, rule := range tb.Config.Rules {
		for ci, condition := range rule.Conditions {

			if len(currNode.Children) == 0 {
				currNode.Children = make(map[string]*rules.Node[T1, T2], 0)
				f, err := tb.SchemaFuncFactory(tb.Config.Schema[ci].Func, tb.Config.Schema[ci].Args)
				if err != nil {
					return err
				}
				currNode.SchemaFunction = f
			}

			_, ok := currNode.Children[condition]
			if ok {
				currNode = currNode.Children[condition]
			} else {
				nextNode := &rules.Node[T1, T2]{}
				currNode.Children[condition] = nextNode
				currNode = nextNode
			}
		}

		for _, res := range rule.Results {
			resFunc, err := tb.ResultFuncFactory(res.Func, res.Args)
			if err != nil {
				return err
			}
			currNode.ResultFunctions = append(currNode.ResultFunctions, resFunc)
		}

		currNode = tree.Root
	}

	return nil
}
