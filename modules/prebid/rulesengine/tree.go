package rulesengine

import (
)

type Tree struct {
	Root Node
}

type Node struct {
	Functions   []Function
	Children    map[string]Node
}
