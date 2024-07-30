package sitter

import "io"

// IterMode indicates the iteration mode.
type IterMode int

// Iterator for a tree of nodes.
type Iterator struct {
	nodesToVisit []*Node
	mode         IterMode
	named        bool
}

// The possible iteration modes.
const (
	DFSMode IterMode = iota
	BFSMode
)

// NewIterator takes a node and mode (DFS/BFS) and returns iterator over children of the node.
func NewIterator(n *Node, mode IterMode) *Iterator {
	return &Iterator{
		named:        false,
		mode:         mode,
		nodesToVisit: []*Node{n},
	}
}

// NewNamedIterator takes a node and mode (DFS/BFS) and returns iterator over named children of the node.
func NewNamedIterator(n *Node, mode IterMode) *Iterator {
	return &Iterator{
		named:        true,
		mode:         mode,
		nodesToVisit: []*Node{n},
	}
}

// Next returns the next node in the current iteration.
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

// ForEach iterates over all nodes, until an error is enconuntered
// (or there are no more nodes).
func (iter *Iterator) ForEach(fn func(*Node) error) (err error) {
	var n *Node

	for {
		if n, err = iter.Next(); err != nil {
			return
		}

		if err = fn(n); err != nil {
			return
		}
	}
}
