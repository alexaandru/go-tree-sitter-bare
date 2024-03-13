package sitter

import "io"

type IterMode int

const (
	DFSMode IterMode = iota
	BFSMode
)

// Iterator for a tree of nodes
type Iterator struct {
	named bool
	mode  IterMode

	nodesToVisit []*Node
}

// NewIterator takes a node and mode (DFS/BFS) and returns iterator over children of the node
func NewIterator(n *Node, mode IterMode) *Iterator {
	return &Iterator{
		named:        false,
		mode:         mode,
		nodesToVisit: []*Node{n},
	}
}

// NewNamedIterator takes a node and mode (DFS/BFS) and returns iterator over named children of the node
func NewNamedIterator(n *Node, mode IterMode) *Iterator {
	return &Iterator{
		named:        true,
		mode:         mode,
		nodesToVisit: []*Node{n},
	}
}

func (iter *Iterator) Next() (n *Node, err error) {
	if len(iter.nodesToVisit) == 0 {
		return nil, io.EOF
	}

	var children []*Node

	n, iter.nodesToVisit = iter.nodesToVisit[0], iter.nodesToVisit[1:]

	if iter.named {
		for i := range int(n.NamedChildCount()) {
			children = append(children, n.NamedChild(i))
		}
	} else {
		for i := range int(n.ChildCount()) {
			children = append(children, n.Child(i))
		}
	}

	switch iter.mode {
	case DFSMode:
		iter.nodesToVisit = append(children, iter.nodesToVisit...)
	case BFSMode:
		iter.nodesToVisit = append(iter.nodesToVisit, children...)
	default:
		panic("not implemented")
	}

	return
}

func (iter *Iterator) ForEach(fn func(*Node) error) error {
	for {
		n, err := iter.Next()
		if err != nil {
			return err
		}

		if err = fn(n); err != nil {
			return err
		}
	}
}
