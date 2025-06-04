package rules

import (
	"encoding/json"
	"errors"
)

// Node represents a node in the tree structure.
// It contains a schema function, a list of result functions, and a map of child nodes.
type Node[T1 any, T2 any] struct {
	SchemaFunction  SchemaFunction[T1]
	ResultFunctions []ResultFunction[T1, T2]
	Children        map[string]*Node[T1, T2]
}

// isLeaf checks if the node is a leaf node.
func (n *Node[T1, T2]) isLeaf() bool {
	return len(n.Children) == 0
}

// matchingChild checks if the node has a child that matches the given value.
// It first checks for an exact match, and if not found, it checks for a wildcard match returning the child node.
// If no matching child is found, it returns nil.
func (n *Node[T1, T2]) matchChild(value string) (string, *Node[T1, T2]) {
	if child, ok := n.Children[value]; ok {
		return value, child
	}
	if child, ok := n.Children["*"]; ok {
		return "*", child
	}
	return "", nil
}

// Tree represents the tree structure.
// It contains a root node and a list of default result functions.
// The tree is generic and can work with any types T1 and T2.
// The tree is used to traverse the nodes based on the schema function results and execute the result functions.
type Tree[T1 any, T2 any] struct {
	Root             *Node[T1, T2]
	DefaultFunctions []ResultFunction[T1, T2]
	AnalyticsKey     string
	ModelVersion     string
}

// Run attempts to walk down the tree from the root to a leaf node. Each node references a schema function
// to execute that returns a result that is used to compare against the node values on the level below it.
// If the result matches one of the node values on the next level, we move to that node, otherwise we exit.
// If a leaf node is reached, it's result functions are executed on the provided result payload.
func (t *Tree[T1, T2]) Run(payload *T1, result *T2) error {
	var nodeKey string
	if t.Root == nil {
		return errors.New("tree root is nil")
	}
	currNode := t.Root

	resFuncMeta := ResultFunctionMeta{
		AnalyticsKey: t.AnalyticsKey,
		ModelVersion: t.ModelVersion,
	}

	for !currNode.isLeaf() {
		if currNode.SchemaFunction == nil {
			return errors.New("schema function is nil")
		}

		res, err := currNode.SchemaFunction.Call(payload)
		if err != nil {
			return err
		}
		resFuncMeta.appendToSchemaFunctionResults(currNode.SchemaFunction.Name(), res)

		nodeKey, currNode = currNode.matchChild(res)
		if currNode == nil {
			resFuncMeta.RuleFired = "default"
			break
		}
		resFuncMeta.appendToRuleFired(nodeKey)
	}

	resultFuncs := t.DefaultFunctions
	if currNode != nil {
		resultFuncs = currNode.ResultFunctions
	}

	for _, rf := range resultFuncs {
		if err := rf.Call(payload, result, resFuncMeta); err != nil {
			return err
		}
	}

	return nil
}

// validate checks if the tree is well-formed which means all leaves are at the same depth.
// It traverses the tree and collects the depths of all leaf nodes and returns an error if
// it finds leaves at different depths.
func (t *Tree[T1, T2]) validate() error {
	if t.Root == nil {
		return nil
	}

	firstLeafDepth := -1

	if !validateNode(t.Root, 0, &firstLeafDepth) {
		return errors.New("tree is malformed: leaves found at different depths")
	}
	return nil
}

// validateNode is a helper function that traverses the tree recursively recording leaf node depths.
func validateNode[T1 any, T2 any](node *Node[T1, T2], depth int, firstLeafDepth *int) bool {
	if node == nil {
		return true
	}

	if node.isLeaf() {
		// If this is the first leaf node we come accross, record
		// its depth
		if *firstLeafDepth == -1 {
			*firstLeafDepth = depth
			return true
		}
		// Else, this is not the first leaf we've visited and depth must
		// match the first leaf's depth. If unequal, tree is unbalanced
		if depth != *firstLeafDepth {
			return false
		}
	}

	for _, child := range node.Children {
		if !validateNode(child, depth+1, firstLeafDepth) {
			return false
		}
	}
	return true
}

// treeBuilder is an interface that defines a method for building a tree.
// It is used to create a tree structure based on the provided configuration.
// The tree builder is expected to implement the Build method which takes a pointer to a Tree
// and returns an error if there is an issue with the configuration or if the tree cannot be built successfully.
// The tree builder is generic and can work with any types T1 and T2.
type treeBuilder[T1 any, T2 any] interface {
	Build(*Tree[T1, T2]) error
}

// NewTree builds a new tree using the provided tree builder function and validates
// the generated tree ensuring it is well-formed.
func NewTree[T1 any, T2 any](builder treeBuilder[T1, T2]) (*Tree[T1, T2], error) {
	tree := Tree[T1, T2]{Root: &Node[T1, T2]{}}

	if err := builder.Build(&tree); err != nil {
		return nil, err
	}

	if err := tree.validate(); err != nil {
		return nil, err
	}

	return &tree, nil
}

// SchemaFuncFactory is a function that takes a function name and arguments in JSON format
// and returns a SchemaFunction and an error.
// It is used to create schema functions for the tree nodes based on the provided configuration.
type SchemaFuncFactory[T any] func(string, json.RawMessage) (SchemaFunction[T], error)

// ResultFuncFactory is a function that takes a function name and arguments in JSON format
// and returns a ResultFunction and an error.
// It is used to create result functions for the tree nodes based on the provided configuration.
type ResultFuncFactory[T1 any, T2 any] func(string, json.RawMessage) (ResultFunction[T1, T2], error)
