package rules

// Create a table driven test for the tree package
import (
	"errors"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestMatchChild(t *testing.T) {
	tests := []struct {
		desc          string
		inNode        *Node[openrtb_ext.RequestWrapper, struct{}]
		inResult      string
		expectedNode  *Node[openrtb_ext.RequestWrapper, struct{}]
		expectedValue string
	}{
		{
			desc: "Childless node",
			inNode: &Node[openrtb_ext.RequestWrapper, struct{}]{
				Children: map[string]*Node[openrtb_ext.RequestWrapper, struct{}]{},
			},
			inResult:      "web",
			expectedNode:  nil,
			expectedValue: "",
		},
		{
			desc: "Result doesn't match and no wildcard",
			inNode: &Node[openrtb_ext.RequestWrapper, struct{}]{
				Children: map[string]*Node[openrtb_ext.RequestWrapper, struct{}]{
					"amp": &Node[openrtb_ext.RequestWrapper, struct{}]{
						SchemaFunction: &deviceCountry{},
					},
				},
			},
			inResult:      "web",
			expectedNode:  nil,
			expectedValue: "",
		},
		{
			desc: "Result doesn't match but node has wildcard",
			inNode: &Node[openrtb_ext.RequestWrapper, struct{}]{
				Children: map[string]*Node[openrtb_ext.RequestWrapper, struct{}]{
					"amp": &Node[openrtb_ext.RequestWrapper, struct{}]{
						SchemaFunction: &deviceCountry{},
					},
					"*": &Node[openrtb_ext.RequestWrapper, struct{}]{
						SchemaFunction: &percent{},
					},
				},
			},
			inResult: "web",
			expectedNode: &Node[openrtb_ext.RequestWrapper, struct{}]{
				SchemaFunction: &percent{},
			},
			expectedValue: "*",
		},
		{
			desc: "Result doesn't match but node has two wildcards",
			inNode: &Node[openrtb_ext.RequestWrapper, struct{}]{
				Children: map[string]*Node[openrtb_ext.RequestWrapper, struct{}]{
					"amp": &Node[openrtb_ext.RequestWrapper, struct{}]{
						SchemaFunction: &deviceCountry{},
					},
					"*": &Node[openrtb_ext.RequestWrapper, struct{}]{
						SchemaFunction: &percent{},
					},
				},
			},
			inResult: "web",
			expectedNode: &Node[openrtb_ext.RequestWrapper, struct{}]{
				SchemaFunction: &percent{},
			},
			expectedValue: "*",
		},
		{
			desc: "Result matches",
			inNode: &Node[openrtb_ext.RequestWrapper, struct{}]{
				Children: map[string]*Node[openrtb_ext.RequestWrapper, struct{}]{
					"web": &Node[openrtb_ext.RequestWrapper, struct{}]{
						SchemaFunction: &deviceCountry{},
					},
					"*": &Node[openrtb_ext.RequestWrapper, struct{}]{
						SchemaFunction: &percent{},
					},
				},
			},
			inResult: "web",
			expectedNode: &Node[openrtb_ext.RequestWrapper, struct{}]{
				SchemaFunction: &deviceCountry{},
			},
			expectedValue: "web",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			resultNode, resultValue := tc.inNode.matchChild(tc.inResult)

			assert.Equal(t, tc.expectedNode, resultNode)
			assert.Equal(t, tc.expectedValue, resultValue)
		})
	}
}

func TestHasEqualDepth(t *testing.T) {
	testCases := []struct {
		desc        string
		inTree      *Tree[openrtb_ext.RequestWrapper, struct{}]
		expectedErr error
	}{
		{
			desc:        "nil root",
			inTree:      &Tree[openrtb_ext.RequestWrapper, struct{}]{},
			expectedErr: nil,
		},
		{
			desc: "leaf only",
			inTree: &Tree[openrtb_ext.RequestWrapper, struct{}]{
				Root: &Node[openrtb_ext.RequestWrapper, struct{}]{
					SchemaFunction: &deviceCountry{},
				},
			},
			expectedErr: nil,
		},
		{
			desc: "Unbalanced tree",
			inTree: &Tree[openrtb_ext.RequestWrapper, struct{}]{
				Root: &Node[openrtb_ext.RequestWrapper, struct{}]{
					SchemaFunction: &deviceCountry{},
					Children: map[string]*Node[openrtb_ext.RequestWrapper, struct{}]{
						"amp": &Node[openrtb_ext.RequestWrapper, struct{}]{
							SchemaFunction: &deviceCountry{},
						},
						"web": &Node[openrtb_ext.RequestWrapper, struct{}]{
							SchemaFunction: &percent{},
							Children: map[string]*Node[openrtb_ext.RequestWrapper, struct{}]{
								"true": &Node[openrtb_ext.RequestWrapper, struct{}]{
									SchemaFunction: &eidIn{},
								},
							},
						},
					},
				},
			},
			expectedErr: errors.New("tree is malformed: leaves found at different depths"),
		},
		{
			desc: "Balanced tree",
			inTree: &Tree[openrtb_ext.RequestWrapper, struct{}]{
				Root: &Node[openrtb_ext.RequestWrapper, struct{}]{
					SchemaFunction: &deviceCountry{},
					Children: map[string]*Node[openrtb_ext.RequestWrapper, struct{}]{
						"amp": &Node[openrtb_ext.RequestWrapper, struct{}]{
							SchemaFunction: &deviceCountry{},
							Children: map[string]*Node[openrtb_ext.RequestWrapper, struct{}]{
								"true": &Node[openrtb_ext.RequestWrapper, struct{}]{
									SchemaFunction: &channel{},
								},
							},
						},
						"web": &Node[openrtb_ext.RequestWrapper, struct{}]{
							SchemaFunction: &percent{},
							Children: map[string]*Node[openrtb_ext.RequestWrapper, struct{}]{
								"true": &Node[openrtb_ext.RequestWrapper, struct{}]{
									SchemaFunction: &eidIn{},
								},
							},
						},
					},
				},
			},
			expectedErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.expectedErr, tc.inTree.validate())
		})
	}
}

// write a test for the Run function
func TestRun(t *testing.T) {
	tests := []struct {
		desc        string
		inTree      *Tree[openrtb_ext.RequestWrapper, struct{}]
		inPayload   *openrtb_ext.RequestWrapper
		expectedErr error
	}{
		{
			desc:   "Nil tree.Root",
			inTree: &Tree[openrtb_ext.RequestWrapper, struct{}]{},
			inPayload: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{},
				},
			},
			expectedErr: errors.New("tree root is nil"),
		},
		{
			desc: "Empty tree",
			inTree: &Tree[openrtb_ext.RequestWrapper, struct{}]{
				Root:         &Node[openrtb_ext.RequestWrapper, struct{}]{},
				AnalyticsKey: "anyAnalyticsKey",
				ModelVersion: "anyModelVersion",
			},
			inPayload: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{},
				},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			result := &struct{}{}
			err := tc.inTree.Run(tc.inPayload, result)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
