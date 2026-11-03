package rules

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLeaf(t *testing.T) {
	tests := []struct {
		name     string
		inNode   *Node[struct{}, struct{}]
		expected bool
	}{
		{
			name:     "Node_with_nil_children",
			inNode:   &Node[struct{}, struct{}]{},
			expected: true,
		},
		{
			name: "Node_with_empty_children_map",
			inNode: &Node[struct{}, struct{}]{
				Children: map[string]*Node[struct{}, struct{}]{},
			},
			expected: true,
		},
		{
			name: "Node_with_children",
			inNode: &Node[struct{}, struct{}]{
				Children: map[string]*Node[struct{}, struct{}]{
					"child": {},
				},
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.inNode.isLeaf())
		})
	}
}

func TestMatchChild(t *testing.T) {
	tests := []struct {
		name            string
		inNode          *Node[struct{}, struct{}]
		nodeKey         string
		expectedNodeKey string
	}{
		{
			name: "nil_children_map",
			inNode: &Node[struct{}, struct{}]{
				Children: nil,
			},
			nodeKey:         "child-one",
			expectedNodeKey: "",
		},
		{
			name: "Childless_node",
			inNode: &Node[struct{}, struct{}]{
				Children: map[string]*Node[struct{}, struct{}]{},
			},
			nodeKey:         "child-one",
			expectedNodeKey: "",
		},
		{
			name: "Result_doesn't_match_and_no_wildcard",
			inNode: &Node[struct{}, struct{}]{
				Children: map[string]*Node[struct{}, struct{}]{
					"child-two": {},
				},
			},
			nodeKey:         "child-one",
			expectedNodeKey: "",
		},
		{
			name: "Result_doesn't_match_but_node_has_wildcard",
			inNode: &Node[struct{}, struct{}]{
				Children: map[string]*Node[struct{}, struct{}]{
					"child-two": {},
					"*":         {},
				},
			},
			nodeKey:         "child-one",
			expectedNodeKey: "*",
		},
		{
			name: "Key_matches",
			inNode: &Node[struct{}, struct{}]{
				Children: map[string]*Node[struct{}, struct{}]{
					"child-one": {},
					"*":         {},
				},
			},
			nodeKey:         "child-one",
			expectedNodeKey: "child-one",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nodeKey, _ := tc.inNode.matchChild(tc.nodeKey)

			assert.Equal(t, tc.expectedNodeKey, nodeKey)
		})
	}
}

func TestTreeValidate(t *testing.T) {
	testCases := []struct {
		name        string
		inTree      *Tree[struct{}, struct{}]
		expectedErr error
	}{
		{
			name:        "nil_root",
			inTree:      &Tree[struct{}, struct{}]{},
			expectedErr: nil,
		},
		{
			name: "root_is_leaf",
			inTree: &Tree[struct{}, struct{}]{
				Root: &Node[struct{}, struct{}]{},
			},
			expectedErr: nil,
		},
		{
			name:        "Unbalanced_tree",
			inTree:      unbalancedTree,
			expectedErr: errors.New("tree is malformed: leaves found at different depths"),
		},
		{
			name:        "Balanced_tree",
			inTree:      balancedTree,
			expectedErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.inTree.validate()
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

var unbalancedTree = &Tree[struct{}, struct{}]{
	Root: &Node[struct{}, struct{}]{
		Children: map[string]*Node[struct{}, struct{}]{
			"child1": {
				Children: map[string]*Node[struct{}, struct{}]{
					"child1.1": {
						Children: map[string]*Node[struct{}, struct{}]{
							"child1.1.1": {},
						},
					},
				},
			},
			"child2": {
				Children: map[string]*Node[struct{}, struct{}]{
					"child2.1": {},
					"child2.2": {},
				},
			},
		},
	},
}

var balancedTree = &Tree[struct{}, struct{}]{
	Root: &Node[struct{}, struct{}]{
		Children: map[string]*Node[struct{}, struct{}]{
			"child1": {
				Children: map[string]*Node[struct{}, struct{}]{
					"child1.1": {},
				},
			},
			"child2": {
				Children: map[string]*Node[struct{}, struct{}]{
					"child2.1": {},
					"child2.2": {},
				},
			},
		},
	},
}

func TestNewTree(t *testing.T) {
	tests := []struct {
		name          string
		inTreeBuilder treeBuilder[struct{}, struct{}]
		expectedTree  *Tree[struct{}, struct{}]
		expectedErr   error
	}{
		{
			name:          "tree_builder_error",
			inTreeBuilder: &faultyTreeBuilder{},
			expectedTree:  nil,
			expectedErr:   errors.New("tree builder error"),
		},
		{
			name:          "Built_tree_is_invalid",
			inTreeBuilder: &builderOfUnbalancedTrees{},
			expectedTree:  nil,
			expectedErr:   errors.New("tree is malformed: leaves found at different depths"),
		},
		{
			name:          "Success",
			inTreeBuilder: &builderOfBalancedTrees{},
			expectedTree:  balancedTree,
			expectedErr:   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tree, err := NewTree(tc.inTreeBuilder)
			assert.Equal(t, tc.expectedTree, tree)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

type faultyTreeBuilder struct{}

func (tb *faultyTreeBuilder) Build(tree *Tree[struct{}, struct{}]) error {
	return errors.New("tree builder error")
}

type builderOfUnbalancedTrees struct{}

func (tb *builderOfUnbalancedTrees) Build(tree *Tree[struct{}, struct{}]) error {
	tree.Root = unbalancedTree.Root
	return nil
}

type builderOfBalancedTrees struct{}

func (tb *builderOfBalancedTrees) Build(tree *Tree[struct{}, struct{}]) error {
	tree.Root = balancedTree.Root
	return nil
}

func TestRun(t *testing.T) {
	tests := []struct {
		name            string
		inTree          *Tree[struct{}, runTestAssertableData]
		expectedResults runTestAssertableData
		expectedErr     error
	}{
		{
			name:   "Nil_tree.Root",
			inTree: &Tree[struct{}, runTestAssertableData]{},
			expectedResults: runTestAssertableData{
				modifiableData: "unmodified_data",
			},
			expectedErr: errors.New("tree root is nil"),
		},
		{
			name: "Single-node_tree",
			inTree: &Tree[struct{}, runTestAssertableData]{
				Root: &Node[struct{}, runTestAssertableData]{},
			},
			expectedResults: runTestAssertableData{modifiableData: "unmodified_data"},
			expectedErr:     nil,
		},
		{
			name: "Schema_function_error",
			inTree: &Tree[struct{}, runTestAssertableData]{
				Root: &Node[struct{}, runTestAssertableData]{
					SchemaFunction: &nodeSchemaFunction{},
					Children: map[string]*Node[struct{}, runTestAssertableData]{
						"nodeSchemaResult": {
							SchemaFunction: &faultySchemaFunction{},
							Children: map[string]*Node[struct{}, runTestAssertableData]{
								"leaf": {},
							},
						},
					},
				},
			},
			expectedResults: runTestAssertableData{modifiableData: "unmodified_data"},
			expectedErr:     errors.New("faulty schema function error"),
		},
		{
			name: "Result_function_error",
			inTree: &Tree[struct{}, runTestAssertableData]{
				Root: &Node[struct{}, runTestAssertableData]{
					SchemaFunction: &nodeSchemaFunction{},
					Children: map[string]*Node[struct{}, runTestAssertableData]{
						"nodeSchemaResult": {
							ResultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
								&errorProneResultFunction{},
							},
						},
					},
				},
			},
			expectedResults: runTestAssertableData{
				modifiableData: "unmodified_data",
				schemaFunctionResults: []SchemaFunctionStep{
					{FuncName: "nodeSchemaFuncName", FuncResult: "nodeSchemaResult"},
				},
				rulesFired: "nodeSchemaResult",
			},
			expectedErr: errors.New("faulty result function error"),
		},
		{
			name: "Schema_return_value_not_matching_any_child_node_default_functions_not_run",
			inTree: &Tree[struct{}, runTestAssertableData]{
				Root: &Node[struct{}, runTestAssertableData]{
					SchemaFunction: &nodeSchemaFunction{},
					Children: map[string]*Node[struct{}, runTestAssertableData]{
						"unreachable-child": {
							ResultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
								&leafResultFunction{},
							},
						},
					},
				},
			},
			expectedResults: runTestAssertableData{modifiableData: "unmodified_data"},
			expectedErr:     nil,
		},
		{
			name: "Schema_return_value_matches_leaf_and_leaf_result_function_is_executed",
			inTree: &Tree[struct{}, runTestAssertableData]{
				Root: &Node[struct{}, runTestAssertableData]{
					SchemaFunction: &nodeSchemaFunction{},
					Children: map[string]*Node[struct{}, runTestAssertableData]{
						"nodeSchemaResult": {
							ResultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
								&leafResultFunction{},
							},
						},
						"*":           {},
						"other-child": {},
					},
				},
			},
			expectedResults: runTestAssertableData{
				modifiableData: "modified_by_leaf_result_function",
				schemaFunctionResults: []SchemaFunctionStep{
					{FuncName: "nodeSchemaFuncName", FuncResult: "nodeSchemaResult"},
				},
				rulesFired: "nodeSchemaResult",
			},
			expectedErr: nil,
		},
		{
			name: "Schema_return_value_not_found,_but_wildcard_exists",
			inTree: &Tree[struct{}, runTestAssertableData]{
				Root: &Node[struct{}, runTestAssertableData]{
					SchemaFunction: &nodeSchemaFunction{},
					Children: map[string]*Node[struct{}, runTestAssertableData]{
						"*": {
							ResultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
								&leafResultFunction{},
							},
						},
					},
				},
			},
			expectedResults: runTestAssertableData{
				modifiableData: "modified_by_leaf_result_function",
				schemaFunctionResults: []SchemaFunctionStep{
					{FuncName: "nodeSchemaFuncName", FuncResult: "nodeSchemaResult"},
				},
				rulesFired: "*",
			},
			expectedErr: nil,
		},
		{
			name: "Counldn't_reach_leaf,_no_default_functions",
			inTree: &Tree[struct{}, runTestAssertableData]{
				Root: &Node[struct{}, runTestAssertableData]{
					SchemaFunction: &nodeSchemaFunction{},
					Children: map[string]*Node[struct{}, runTestAssertableData]{
						"key-different-from-schema-result": {
							ResultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
								&leafResultFunction{},
							},
						},
					},
				},
			},
			expectedResults: runTestAssertableData{modifiableData: "unmodified_data"},
			expectedErr:     nil,
		},
		{
			name: "Couldn't_reach_leaf,_run_default_functions",
			inTree: &Tree[struct{}, runTestAssertableData]{
				DefaultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
					&defaultResultFunction{},
				},
				Root: &Node[struct{}, runTestAssertableData]{
					SchemaFunction: &nodeSchemaFunction{},
					Children: map[string]*Node[struct{}, runTestAssertableData]{
						"key-different-from-schema-result": {
							ResultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
								&leafResultFunction{},
							},
						},
					},
				},
			},
			expectedResults: runTestAssertableData{
				modifiableData: "modified_by_default_function",
				schemaFunctionResults: []SchemaFunctionStep{
					{FuncName: "nodeSchemaFuncName", FuncResult: "nodeSchemaResult"},
				},
				rulesFired: "default",
			},
			expectedErr: nil,
		},
		{
			name: "Leaf_found_without_result_functions_do_not_execute_defaults",
			inTree: &Tree[struct{}, runTestAssertableData]{
				Root: &Node[struct{}, runTestAssertableData]{
					SchemaFunction: &nodeSchemaFunction{},
					Children: map[string]*Node[struct{}, runTestAssertableData]{
						"nodeSchemaResult": {
							ResultFunctions: []ResultFunction[struct{}, runTestAssertableData]{},
						},
					},
				},
				DefaultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
					&defaultResultFunction{},
				},
			},
			expectedResults: runTestAssertableData{modifiableData: "unmodified_data"},
			expectedErr:     nil,
		},
		{
			name: "Missing_schema_function",
			inTree: &Tree[struct{}, runTestAssertableData]{
				Root: &Node[struct{}, runTestAssertableData]{
					SchemaFunction: nil,
					Children: map[string]*Node[struct{}, runTestAssertableData]{
						"nodeSchemaResult": {
							ResultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
								&leafResultFunction{},
							},
						},
					},
				},
			},
			expectedResults: runTestAssertableData{modifiableData: "unmodified_data"},
			expectedErr:     errors.New("schema function is nil"),
		},
		{
			name: "Leaf_found._Run_multiple_leaf_result_functions",
			inTree: &Tree[struct{}, runTestAssertableData]{
				Root: &Node[struct{}, runTestAssertableData]{
					SchemaFunction: &nodeSchemaFunction{},
					Children: map[string]*Node[struct{}, runTestAssertableData]{
						"nodeSchemaResult": {
							ResultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
								&leafResultFunction{},
								&leafResultFunction{},
								&leafResultFunction{},
							},
						},
					},
				},
			},
			expectedResults: runTestAssertableData{
				modifiableData: "modified_by_leaf_result_function-modified_by_leaf_result_function-modified_by_leaf_result_function",
				schemaFunctionResults: []SchemaFunctionStep{
					{FuncName: "nodeSchemaFuncName", FuncResult: "nodeSchemaResult"},
				},
				rulesFired: "nodeSchemaResult",
			},
			expectedErr: nil,
		},
		{
			name: "Couldn't_reach_leaf,_run_multiple_default_functions",
			inTree: &Tree[struct{}, runTestAssertableData]{
				DefaultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
					&defaultResultFunction{},
					&defaultResultFunction{},
					&defaultResultFunction{},
				},
				Root: &Node[struct{}, runTestAssertableData]{
					SchemaFunction: &nodeSchemaFunction{},
					Children: map[string]*Node[struct{}, runTestAssertableData]{
						"unreachable-leaf": {
							ResultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
								&leafResultFunction{},
							},
						},
					},
				},
			},
			expectedResults: runTestAssertableData{
				modifiableData: "modified_by_default_function-modified_by_default_function-modified_by_default_function",
				schemaFunctionResults: []SchemaFunctionStep{
					{FuncName: "nodeSchemaFuncName", FuncResult: "nodeSchemaResult"},
				},
				rulesFired: "default",
			},
			expectedErr: nil,
		},
		{
			name: "Reach_leaf_multiple_levels_down",
			inTree: &Tree[struct{}, runTestAssertableData]{
				DefaultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
					&defaultResultFunction{},
				},
				Root: &Node[struct{}, runTestAssertableData]{
					SchemaFunction: &nodeSchemaFunction{},
					Children: map[string]*Node[struct{}, runTestAssertableData]{
						"key-different-from-schema-result": {},
						"nodeSchemaResult": {
							SchemaFunction: &nodeSchemaFunction{},
							Children: map[string]*Node[struct{}, runTestAssertableData]{
								"*": {
									SchemaFunction: &nodeSchemaFunction{},
									Children: map[string]*Node[struct{}, runTestAssertableData]{
										"nodeSchemaResult": {
											ResultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
												&leafResultFunction{},
												&leafResultFunction{},
											},
										},
									},
								},
								"key-different-from-schema-result": {
									SchemaFunction: &nodeSchemaFunction{},
									Children: map[string]*Node[struct{}, runTestAssertableData]{
										"unreachable-leaf": {
											ResultFunctions: []ResultFunction[struct{}, runTestAssertableData]{
												&leafResultFunction{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedResults: runTestAssertableData{
				modifiableData: "modified_by_leaf_result_function-modified_by_leaf_result_function",
				schemaFunctionResults: []SchemaFunctionStep{
					{FuncName: "nodeSchemaFuncName", FuncResult: "nodeSchemaResult"},
					{FuncName: "nodeSchemaFuncName", FuncResult: "nodeSchemaResult"},
					{FuncName: "nodeSchemaFuncName", FuncResult: "nodeSchemaResult"},
				},
				rulesFired: "nodeSchemaResult|*|nodeSchemaResult",
			},
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			anyPayload := struct{}{}
			result := runTestAssertableData{modifiableData: "unmodified_data"}

			err := tc.inTree.Run(&anyPayload, &result)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedResults, result)
		})
	}
}

// helper schema functions
type nodeSchemaFunction struct{}

func (sf *nodeSchemaFunction) Call(param *struct{}) (string, error) {
	return "nodeSchemaResult", nil
}
func (sf *nodeSchemaFunction) Name() string {
	return "nodeSchemaFuncName"
}

type faultySchemaFunction struct{}

func (sf *faultySchemaFunction) Call(param *struct{}) (string, error) {
	return "", errors.New("faulty schema function error")
}
func (sf *faultySchemaFunction) Name() string {
	return "faultySchemaFunction"
}

// helper result functions
type runTestAssertableData struct {
	modifiableData        string
	schemaFunctionResults []SchemaFunctionStep
	rulesFired            string
}

type leafResultFunction struct{}

func (sf *leafResultFunction) Call(param *struct{}, modifiables *runTestAssertableData, meta ResultFunctionMeta) error {
	if len(modifiables.schemaFunctionResults) == 0 {
		modifiables.schemaFunctionResults = append(modifiables.schemaFunctionResults, meta.SchemaFunctionResults...)
	}
	if len(modifiables.rulesFired) == 0 {
		modifiables.rulesFired = meta.RuleFired
	}
	if modifiables.modifiableData == "unmodified_data" {
		modifiables.modifiableData = "modified_by_leaf_result_function"
	} else {
		modifiables.modifiableData = fmt.Sprintf("%s-modified_by_leaf_result_function", modifiables.modifiableData)
	}
	return nil
}
func (sf *leafResultFunction) Name() string {
	return "leafResultFunction"
}

type defaultResultFunction struct{}

func (sf *defaultResultFunction) Call(param *struct{}, modifiables *runTestAssertableData, meta ResultFunctionMeta) error {
	if len(modifiables.schemaFunctionResults) == 0 {
		modifiables.schemaFunctionResults = append(modifiables.schemaFunctionResults, meta.SchemaFunctionResults...)
	}
	if len(modifiables.rulesFired) == 0 {
		modifiables.rulesFired = meta.RuleFired
	}
	if modifiables.modifiableData == "unmodified_data" {
		modifiables.modifiableData = "modified_by_default_function"
	} else {
		modifiables.modifiableData = fmt.Sprintf("%s-modified_by_default_function", modifiables.modifiableData)
	}
	return nil
}
func (sf *defaultResultFunction) Name() string {
	return "defaultResultFunction"
}

type errorProneResultFunction struct{}

func (sf *errorProneResultFunction) Call(param *struct{}, modifiables *runTestAssertableData, meta ResultFunctionMeta) error {
	if len(modifiables.schemaFunctionResults) == 0 {
		modifiables.schemaFunctionResults = append(modifiables.schemaFunctionResults, meta.SchemaFunctionResults...)
	}
	if len(modifiables.rulesFired) == 0 {
		modifiables.rulesFired = meta.RuleFired
	}
	return errors.New("faulty result function error")
}
func (sf *errorProneResultFunction) Name() string {
	return "faultyResultFunction"
}
