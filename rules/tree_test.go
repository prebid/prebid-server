package rules

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchChild(t *testing.T) {
	tests := []struct {
		desc            string
		inNode          *Node[struct{}, struct{}]
		nodeKey         string
		expectedNodeKey string
	}{
		{
			desc: "nil children map",
			inNode: &Node[struct{}, struct{}]{
				Children: nil,
			},
			nodeKey:         "child-one",
			expectedNodeKey: "",
		},
		{
			desc: "Childless node",
			inNode: &Node[struct{}, struct{}]{
				Children: map[string]*Node[struct{}, struct{}]{},
			},
			nodeKey:         "child-one",
			expectedNodeKey: "",
		},
		{
			desc: "Result doesn't match and no wildcard",
			inNode: &Node[struct{}, struct{}]{
				Children: map[string]*Node[struct{}, struct{}]{
					"child-two": &Node[struct{}, struct{}]{},
				},
			},
			nodeKey:         "child-one",
			expectedNodeKey: "",
		},
		{
			desc: "Result doesn't match but node has wildcard",
			inNode: &Node[struct{}, struct{}]{
				Children: map[string]*Node[struct{}, struct{}]{
					"child-two": &Node[struct{}, struct{}]{},
					"*":         &Node[struct{}, struct{}]{},
				},
			},
			nodeKey:         "child-one",
			expectedNodeKey: "*",
		},
		{
			desc: "Key matches",
			inNode: &Node[struct{}, struct{}]{
				Children: map[string]*Node[struct{}, struct{}]{
					"child-one": &Node[struct{}, struct{}]{},
					"*":         &Node[struct{}, struct{}]{},
				},
			},
			nodeKey:         "child-one",
			expectedNodeKey: "child-one",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			nodeKey, _ := tc.inNode.matchChild(tc.nodeKey)

			assert.Equal(t, tc.expectedNodeKey, nodeKey)
		})
	}
}

func TestTreeValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		inTree      *Tree[struct{}, struct{}]
		expectedErr error
	}{
		{
			desc:        "nil root",
			inTree:      &Tree[struct{}, struct{}]{},
			expectedErr: nil,
		},
		{
			desc: "leaf only",
			inTree: &Tree[struct{}, struct{}]{
				Root: &Node[struct{}, struct{}]{},
			},
			expectedErr: nil,
		},
		{
			desc: "Unbalanced tree",
			inTree: &Tree[struct{}, struct{}]{
				Root: &Node[struct{}, struct{}]{
					Children: map[string]*Node[struct{}, struct{}]{
						"child1": &Node[struct{}, struct{}]{},
						"child2": &Node[struct{}, struct{}]{
							Children: map[string]*Node[struct{}, struct{}]{
								"child2.1": &Node[struct{}, struct{}]{},
							},
						},
					},
				},
			},
			expectedErr: errors.New("tree is malformed: leaves found at different depths"),
		},
		{
			desc: "Balanced tree",
			inTree: &Tree[struct{}, struct{}]{
				Root: &Node[struct{}, struct{}]{
					Children: map[string]*Node[struct{}, struct{}]{
						"child1": &Node[struct{}, struct{}]{
							Children: map[string]*Node[struct{}, struct{}]{
								"child1.1": &Node[struct{}, struct{}]{}},
						},
						"child2": &Node[struct{}, struct{}]{
							Children: map[string]*Node[struct{}, struct{}]{
								"child2.1": &Node[struct{}, struct{}]{},
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

func TestRun(t *testing.T) {
	tests := []struct {
		desc                 string
		inTree               *Tree[struct{}, string]
		inModifiableData     string
		expectedModifiedData string
		expectedErr          error
	}{
		{
			desc:                 "Nil tree.Root",
			inTree:               &Tree[struct{}, string]{},
			inModifiableData:     "str",
			expectedModifiedData: "str",
			expectedErr:          nil,
		},
		{
			desc: "Single-node tree",
			inTree: &Tree[struct{}, string]{
				Root: &Node[struct{}, string]{
					SchemaFunction:  &goodSchemaFunction{},
					ResultFunctions: []ResultFunction[struct{}, string]{},
				},
			},
			inModifiableData:     "str",
			expectedModifiedData: "str",
			expectedErr:          nil,
		},
		{
			desc: "Schema function error",
			inTree: &Tree[struct{}, string]{
				Root: &Node[struct{}, string]{
					SchemaFunction: &faultySchemaFunction{},
					Children: map[string]*Node[struct{}, string]{
						"leaf": &Node[struct{}, string]{},
					},
				},
			},
			inModifiableData:     "str",
			expectedModifiedData: "str",
			expectedErr:          errors.New("faulty schema function error"),
		},
		{
			desc: "Result function error",
			inTree: &Tree[struct{}, string]{
				Root: &Node[struct{}, string]{
					SchemaFunction: &goodSchemaFunction{},
					ResultFunctions: []ResultFunction[struct{}, string]{
						&unexpectedResultFunction{},
					},
					Children: map[string]*Node[struct{}, string]{
						"goodValue": &Node[struct{}, string]{
							ResultFunctions: []ResultFunction[struct{}, string]{
								&faultyResultFunction{},
							},
						},
					},
				},
			},
			inModifiableData:     "str",
			expectedModifiedData: "str",
			expectedErr:          errors.New("faulty result function error"),
		},
		{
			desc: "Schema return value not matching any child node",
			inTree: &Tree[struct{}, string]{
				Root: &Node[struct{}, string]{
					SchemaFunction: &goodSchemaFunction{},
					ResultFunctions: []ResultFunction[struct{}, string]{
						&unexpectedResultFunction{},
					},
					Children: map[string]*Node[struct{}, string]{
						"unreachable-child": &Node[struct{}, string]{
							ResultFunctions: []ResultFunction[struct{}, string]{
								&leafResultFunction{},
							},
						},
					},
				},
			},
			inModifiableData:     "str",
			expectedModifiedData: "str",
			expectedErr:          nil,
		},
		{
			desc: "Schema return value matches child and correct result function is executed",
			inTree: &Tree[struct{}, string]{
				Root: &Node[struct{}, string]{
					SchemaFunction: &goodSchemaFunction{},
					Children: map[string]*Node[struct{}, string]{
						"goodValue": &Node[struct{}, string]{
							SchemaFunction: &goodSchemaFunction{},
							ResultFunctions: []ResultFunction[struct{}, string]{
								&unexpectedResultFunction{},
							},
							Children: map[string]*Node[struct{}, string]{
								"goodValue": &Node[struct{}, string]{
									ResultFunctions: []ResultFunction[struct{}, string]{
										&leafResultFunction{},
									},
								},
								"unreachable-leaf": &Node[struct{}, string]{},
							},
						},
						"*": &Node[struct{}, string]{
							ResultFunctions: []ResultFunction[struct{}, string]{
								&unexpectedResultFunction{},
							},
						},
						"unreachable-child": &Node[struct{}, string]{},
					},
				},
			},
			inModifiableData:     "str",
			expectedModifiedData: "str-modified-by-leaf-result-function",
			expectedErr:          nil,
		},
		{
			desc: "Schema return value not found in children, but wildcard exists",
			inTree: &Tree[struct{}, string]{
				Root: &Node[struct{}, string]{
					SchemaFunction: &goodSchemaFunction{},
					Children: map[string]*Node[struct{}, string]{
						"unreachable-child": &Node[struct{}, string]{},
						"*": &Node[struct{}, string]{
							SchemaFunction: &faultySchemaFunction{},
							ResultFunctions: []ResultFunction[struct{}, string]{
								&leafResultFunction{},
							},
						},
					},
				},
			},
			inModifiableData:     "str",
			expectedModifiedData: "str-modified-by-leaf-result-function",
			expectedErr:          nil,
		},
		{
			desc: "Counldn't reach leaf, no default functions",
			inTree: &Tree[struct{}, string]{
				Root: &Node[struct{}, string]{
					SchemaFunction: &goodSchemaFunction{},
					Children: map[string]*Node[struct{}, string]{
						"goodValue": &Node[struct{}, string]{
							SchemaFunction: &goodSchemaFunction{},
							ResultFunctions: []ResultFunction[struct{}, string]{
								&unexpectedResultFunction{},
							},
							Children: map[string]*Node[struct{}, string]{
								"*": &Node[struct{}, string]{
									SchemaFunction: &goodSchemaFunction{},
									ResultFunctions: []ResultFunction[struct{}, string]{
										&unexpectedResultFunction{},
									},
									Children: map[string]*Node[struct{}, string]{
										"unreachable-leaf": &Node[struct{}, string]{
											ResultFunctions: []ResultFunction[struct{}, string]{
												&unexpectedResultFunction{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			inModifiableData:     "str",
			expectedModifiedData: "str",
			expectedErr:          nil,
		},
		{
			desc: "Counldn't reach leaf, run default functions",
			inTree: &Tree[struct{}, string]{
				Root: &Node[struct{}, string]{
					SchemaFunction: &goodSchemaFunction{},
					Children: map[string]*Node[struct{}, string]{
						"*": &Node[struct{}, string]{
							SchemaFunction: &goodSchemaFunction{},
							ResultFunctions: []ResultFunction[struct{}, string]{
								&unexpectedResultFunction{},
							},
							Children: map[string]*Node[struct{}, string]{
								"unreachable-child": &Node[struct{}, string]{
									SchemaFunction: &goodSchemaFunction{},
									ResultFunctions: []ResultFunction[struct{}, string]{
										&leafResultFunction{},
									},
								},
							},
						},
					},
				},
				DefaultFunctions: []ResultFunction[struct{}, string]{
					&defaultResultFunction{},
				},
			},
			inModifiableData:     "str",
			expectedModifiedData: "str-modified-by-default-function",
			expectedErr:          nil,
		},
		{
			desc: "Leaf contains no result functions, run default functions instead",
			inTree: &Tree[struct{}, string]{
				Root: &Node[struct{}, string]{
					SchemaFunction: &goodSchemaFunction{},
					Children: map[string]*Node[struct{}, string]{
						"*": &Node[struct{}, string]{
							SchemaFunction: &goodSchemaFunction{},
							ResultFunctions: []ResultFunction[struct{}, string]{
								&unexpectedResultFunction{},
							},
							Children: map[string]*Node[struct{}, string]{
								"goodValue": &Node[struct{}, string]{
									SchemaFunction:  &goodSchemaFunction{},
									ResultFunctions: []ResultFunction[struct{}, string]{},
								},
							},
						},
					},
				},
				DefaultFunctions: []ResultFunction[struct{}, string]{
					&defaultResultFunction{},
				},
			},
			inModifiableData:     "str",
			expectedModifiedData: "str",
			expectedErr:          nil,
		},
		{
			desc: "Missing schema function in non-root node",
			inTree: &Tree[struct{}, string]{
				Root: &Node[struct{}, string]{
					SchemaFunction: &goodSchemaFunction{},
					Children: map[string]*Node[struct{}, string]{
						"*": &Node[struct{}, string]{
							SchemaFunction: nil, // Missing schema function
							ResultFunctions: []ResultFunction[struct{}, string]{
								&unexpectedResultFunction{},
							},
							Children: map[string]*Node[struct{}, string]{
								"goodValue": &Node[struct{}, string]{
									SchemaFunction: &goodSchemaFunction{},
									ResultFunctions: []ResultFunction[struct{}, string]{
										&leafResultFunction{},
									},
								},
							},
						},
					},
				},
			},
			inModifiableData:     "str",
			expectedModifiedData: "str",
			expectedErr:          errors.New("schema function is nil"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			anyPayload := struct{}{}
			err := tc.inTree.Run(&anyPayload, &tc.inModifiableData)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedModifiedData, tc.inModifiableData)
		})
	}
}

type goodSchemaFunction struct{}

func (sf *goodSchemaFunction) Call(param *struct{}) (string, error) {
	return "goodValue", nil
}
func (sf *goodSchemaFunction) Name() string {
	return "goodSchemaFunction"
}

type faultySchemaFunction struct{}

func (sf *faultySchemaFunction) Call(param *struct{}) (string, error) {
	return "", errors.New("faulty schema function error")
}
func (sf *faultySchemaFunction) Name() string {
	return "faultySchemaFunction"
}

type leafResultFunction struct{}

func (sf *leafResultFunction) Call(param *struct{}, modifiable *string, meta ResultFunctionMeta) error {
	*modifiable = fmt.Sprintf("%s-modified-by-leaf-result-function", *modifiable)
	return nil
}
func (sf *leafResultFunction) Name() string {
	return "leafResultFunction"
}

type defaultResultFunction struct{}

func (sf *defaultResultFunction) Call(param *struct{}, modifiable *string, meta ResultFunctionMeta) error {
	*modifiable = fmt.Sprintf("%s-modified-by-default-function", *modifiable)
	return nil
}
func (sf *defaultResultFunction) Name() string {
	return "defaultResultFunction9"
}

type unexpectedResultFunction struct{}

func (sf *unexpectedResultFunction) Call(param *struct{}, modifiable *string, meta ResultFunctionMeta) error {
	*modifiable = fmt.Sprintf("%s-wrong-modification", *modifiable)
	return nil
}
func (sf *unexpectedResultFunction) Name() string {
	return "unexpectedResultFunction"
}

type faultyResultFunction struct{}

func (sf *faultyResultFunction) Call(param *struct{}, modifiable *string, meta ResultFunctionMeta) error {
	return errors.New("faulty result function error")
}
func (sf *faultyResultFunction) Name() string {
	return "faultyResultFunction"
}
