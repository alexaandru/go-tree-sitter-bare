package sitter

// #include "bindings.h"
import "C"
import "runtime"

// TreeCursor allows you to walk a syntax tree more efficiently than is
// possible using the `Node` functions. It is a mutable object that is always
// on a certain syntax node, and can be moved imperatively to different nodes.
type TreeCursor struct {
	c *C.TSTreeCursor
	t *Tree

	isClosed bool
}

// NewTreeCursor creates a new tree cursor starting from the given node.
func NewTreeCursor(n *Node) *TreeCursor {
	cc := C.ts_tree_cursor_new(n.c)
	c := &TreeCursor{
		c: &cc,
		t: n.t,
	}

	runtime.SetFinalizer(c, (*TreeCursor).Close)

	return c
}

// Close should be called to ensure that all the memory used by the tree cursor
// is freed.
//
// As the constructor in go-tree-sitter would set this func call through runtime.SetFinalizer,
// parser.Close() will be called by Go's garbage collector and users would not have to call this manually.
func (c *TreeCursor) Close() {
	if !c.isClosed {
		C.ts_tree_cursor_delete(c.c)
	}

	c.isClosed = true
}

// Reset re-initializes a tree cursor to start at a different node.
func (c *TreeCursor) Reset(n *Node) {
	c.t = n.t
	C.ts_tree_cursor_reset(c.c, n.c)
}

// CurrentNode of the tree cursor.
func (c *TreeCursor) CurrentNode() *Node {
	n := C.ts_tree_cursor_current_node(c.c)
	return c.t.cachedNode(n)
}

// CurrentFieldName gets the field name of the tree cursor's current node.
//
// This returns empty string if the current node doesn't have a field.
func (c *TreeCursor) CurrentFieldName() string {
	return C.GoString(C.ts_tree_cursor_current_field_name(c.c))
}

// GoToParent moves the cursor to the parent of its current node.
//
// This returns `true` if the cursor successfully moved, and returns `false`
// if there was no parent node (the cursor was already on the root node).
func (c *TreeCursor) GoToParent() bool {
	return bool(C.ts_tree_cursor_goto_parent(c.c))
}

// GoToNextSibling moves the cursor to the next sibling of its current node.
//
// This returns `true` if the cursor successfully moved, and returns `false`
// if there was no next sibling node.
func (c *TreeCursor) GoToNextSibling() bool {
	return bool(C.ts_tree_cursor_goto_next_sibling(c.c))
}

// GoToFirstChild moves the cursor to the first child of its current node.
//
// This returns `true` if the cursor successfully moved, and returns `false`
// if there were no children.
func (c *TreeCursor) GoToFirstChild() bool {
	return bool(C.ts_tree_cursor_goto_first_child(c.c))
}

// GoToFirstChildForByte moves the cursor to the first child of its current node
// that extends beyond the given byte offset.
//
// This returns the index of the child node if one was found, and returns -1
// if no such child was found.
func (c *TreeCursor) GoToFirstChildForByte(b uint32) int64 {
	return int64(C.ts_tree_cursor_goto_first_child_for_byte(c.c, C.uint32_t(b)))
}
