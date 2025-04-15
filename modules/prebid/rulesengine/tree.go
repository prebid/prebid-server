package rulesengine

type Tree struct {
	Root Node
}

type Node struct {
	Functions []Function
	Children  map[string]Node
}
