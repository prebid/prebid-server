package optimizationmodule

import (
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type Rules struct {
	Root *Node
}

type Node struct {
	Function Function
	// string because  "rules": [{ "conditions": ["true", "false"],
	Children map[string]*Node
}

func (r *Rules) Execute(rw *openrtb_ext.RequestWrapper) error {
	_, err := r.Root.ProcessNode(rw)
	return err
}

func (n *Node) ProcessNode(rw *openrtb_ext.RequestWrapper) (string, error) {
	res, err := n.Function.Call(rw)
	if len(n.Children) == 0 {
		return res, err
	}
	next := n.Children[res]
	if next == nil {
		return res, err
	}
	return next.ProcessNode(rw)
}

func ExecuteFlat(r *Rules, rw *openrtb_ext.RequestWrapper) (string, error) {
	currNode := r.Root

	for len(currNode.Children) > 0 {
		// schema function
		res, _ := currNode.Function.Call(rw)

		next := currNode.Children[res]

		if next != nil {
			currNode = next
		}
	}
	// result function, should be a list of func
	res, err := currNode.Function.Call(rw)
	return res, err

}
