package sitter

// #include "bindings.h"
import "C"
import "runtime"

// BaseTree is needed as we use cache for nodes on normal tree object.
// It prevent run of SetFinalizer as it introduces cycle we can workaround it using
// separate object for details see: https://github.com/golang/go/issues/7358#issuecomment-66091558
type BaseTree struct {
	c        *C.TSTree
	isClosed bool
}

// Tree represents the syntax tree of an entire source code file
// Note: Tree instances are not thread safe;
// you must copy a tree if you want to use it on multiple threads simultaneously.
type Tree struct {
	*BaseTree

	// p is a pointer to a Parser that produced the Tree. Only used to keep Parser alive.
	// Otherwise Parser may be GC'ed (and deleted by the finalizer) while some Tree objects are still in use.
	p *Parser

	// most probably better save node.id
	cache map[C.TSNode]*Node
}

// Point represents one location in the input.
type Point struct {
	Row    uint32
	Column uint32
}

// Range represents a range in the input.
type Range struct {
	StartPoint Point
	EndPoint   Point
	StartByte  uint32
	EndByte    uint32
}

// EditInput represents one edit in the input.
type EditInput struct {
	StartIndex  uint32
	OldEndIndex uint32
	NewEndIndex uint32
	StartPoint  Point
	OldEndPoint Point
	NewEndPoint Point
}

func (i EditInput) c() *C.TSInputEdit {
	return &C.TSInputEdit{
		start_byte:   C.uint32_t(i.StartIndex),
		old_end_byte: C.uint32_t(i.OldEndIndex),
		new_end_byte: C.uint32_t(i.NewEndIndex),
		start_point: C.TSPoint{
			row:    C.uint32_t(i.StartPoint.Row),
			column: C.uint32_t(i.StartPoint.Column),
		},
		old_end_point: C.TSPoint{
			row:    C.uint32_t(i.OldEndPoint.Row),
			column: C.uint32_t(i.OldEndPoint.Column),
		},
		new_end_point: C.TSPoint{
			row:    C.uint32_t(i.OldEndPoint.Row),
			column: C.uint32_t(i.OldEndPoint.Column),
		},
	}
}

// newTree creates a new tree object from a C pointer. The function will set a finalizer for the object,
// thus no free is needed for it.
func (p *Parser) newTree(c *C.TSTree) *Tree {
	base := &BaseTree{c: c}

	runtime.SetFinalizer(base, (*BaseTree).Close)

	newTree := &Tree{p: p, BaseTree: base, cache: map[C.TSNode]*Node{}}

	return newTree
}

// Edit the syntax tree to keep it in sync with source code that has been edited.
func (t *Tree) Edit(i EditInput) {
	C.ts_tree_edit(t.c, i.c())
}

// Copy returns a new copy of a tree
func (t *Tree) Copy() *Tree {
	return t.p.newTree(C.ts_tree_copy(t.c))
}

// RootNode returns root node of a tree
func (t *Tree) RootNode() *Node {
	ptr := C.ts_tree_root_node(t.c)

	return t.cachedNode(ptr)
}

func (t *Tree) cachedNode(ptr C.TSNode) *Node {
	if ptr.id == nil {
		return nil
	}

	if n, ok := t.cache[ptr]; ok {
		return n
	}

	n := &Node{ptr, t}
	t.cache[ptr] = n

	return n
}

// Close should be called to ensure that all the memory used by the tree is freed.
//
// As the constructor in go-tree-sitter would set this func call through runtime.SetFinalizer,
// parser.Close() will be called by Go's garbage collector and users would not have to call this manually.
func (t *BaseTree) Close() {
	if !t.isClosed {
		C.ts_tree_delete(t.c)
	}

	t.isClosed = true
}
