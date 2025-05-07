package rules

import (
	"encoding/json"
)

// Node...
type Node [T1 any, T2 any] struct {
	SchemaFunction  SchemaFunction[T1]
	ResultFunctions []ResultFunction[T2]
	Children        map[string]*Node[T1, T2]
}

// Tree...
type Tree [T1 any, T2 any] struct {
	Root *Node[T1, T2]
}

// Run attempts to walk down the tree from the root to a leaf node. Each node references a schema function
// to execute that returns a result that is used to compare against the node values on the level below it.
// If the result matches one of the node values on the next level, we move to that node, otherwise we exit.
// If a leaf node is reached, it's result functions are executed on the provided result payload.
func(t *Tree[T1, T2]) Run(payload *T1, result *T2) error {
	currNode := t.Root

	for len(currNode.Children) > 0 {
		res, err := currNode.SchemaFunction.Call(payload)
		if err != nil {
			return err
		}

		next := currNode.Children[res]
		if next != nil {
			currNode = next
		}
	}

	// TODO: handle default - does that belong here or in the builder function?
	// if !currNode.IsLeaf() {
		// should we put default result functions on every node or on the tree?
	// }

	for _, rf := range currNode.ResultFunctions {
		err := rf.Call(result)
		if err != nil {
			return err
		}
	}
	return nil
}

// Valid ensures that the tree is well-formed meaning that every leaf is at the same level
func(t *Tree[T1, T2]) validate() error {
	//TODO
	return nil
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
type ResultFuncFactory[T any] func(string, json.RawMessage) (ResultFunction[T], error)