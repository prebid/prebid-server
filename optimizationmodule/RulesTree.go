package optimizationmodule

import (
	"encoding/json"
	"fmt"
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

func Build(config json.RawMessage) Rules {
	//parse config json

	//stub for prototype
	rules := Rules{
		Root: &Node{
			Function: NewDeviceCountry([]string{"USA"}),
			Children: map[string]*Node{
				"yes": &Node{Function: NewSetDevIp([]string{"127.0.0.1"})}, //can have children
				"no":  &Node{Function: NewSetDevIp([]string{"127.0.0.2"})}, // can have children
			},
		},
	}
	return rules
}

func (r *Rules) Execute(rw *openrtb_ext.RequestWrapper) error {
	res, err := r.Root.ProcessNode(rw)
	fmt.Println(res)
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
