package sitter

import "io"

// IterMode indicates the iteration mode.
type IterMode int

// Iterator for a tree of nodes.
type Iterator struct {
	toVisit []*Node
	mode    IterMode
	named   bool
}

// The possible iteration modes.
const (
	DFSMode IterMode = iota
	BFSMode
)

// NewIterator takes a node and mode (DFS/BFS) and returns iterator over
// children of the node.
func NewIterator(n *Node, mode IterMode) *Iterator {
	return &Iterator{toVisit: []*Node{n}, mode: mode}
}

// NewNamedIterator takes a node and mode (DFS/BFS) and returns iterator
// over named children of the node.
func NewNamedIterator(n *Node, mode IterMode) *Iterator {
	return &Iterator{toVisit: []*Node{n}, mode: mode, named: true}
}

// Next returns the next node in the current iteration.
func (iter *Iterator) Next() (n *Node, err error) {
	if len(iter.toVisit) == 0 {
		return nil, io.EOF
	}

	var children []*Node

	n, iter.toVisit = iter.toVisit[0], iter.toVisit[1:]

	if iter.named {
		for i := range n.NamedChildCount() {
			children = append(children, n.NamedChild(i))
		}
	} else {
		for i := range n.ChildCount() {
			children = append(children, n.Child(i))
		}
	}

	switch iter.mode {
	case DFSMode:
		iter.toVisit = append(children, iter.toVisit...)
	case BFSMode:
		iter.toVisit = append(iter.toVisit, children...)
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
