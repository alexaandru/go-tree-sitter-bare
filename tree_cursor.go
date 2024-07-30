package sitter

// #include "sitter.h"
import "C"
import "runtime"

// TreeCursor allows you to walk a syntax tree more efficiently than is
// possible using the `Node` functions. It is a mutable object that is always
// on a certain syntax node, and can be moved imperatively to different nodes.
type TreeCursor struct {
	c       *C.TSTreeCursor
	t       *Tree
	context [3]uint32 // TODO: How is this used upstream?

	isClosed bool
}

// NewTreeCursor creates a new tree cursor starting from the given node.
//
// A tree cursor allows you to walk a syntax tree more efficiently than is
// possible using the [`TSNode`] functions. It is a mutable object that is always
// on a certain syntax node, and can be moved imperatively to different nodes.
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

// ResetTo re-initializes a tree cursor to the same position as another cursor.
//
// Unlike `ts_tree_cursor_reset`, this will not lose parent information and
// allows reusing already created cursors.
func (c *TreeCursor) ResetTo(src *TreeCursor) {
	C.ts_tree_cursor_reset_to(c.c, src.c)
}

// CurrentNode returns the cursor's current node.
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

// CurrentFieldID returns the field id of the tree cursor's current node.
//
// This returns zero if the current node doesn't have a field.
// See also [`ts_node_child_by_field_id`], [`ts_language_field_id_for_name`].
func (c *TreeCursor) CurrentFieldID() FieldID {
	return C.ts_tree_cursor_current_field_id(c.c)
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

// GotoPreviousSibling moves the cursor to the previous sibling of its current node.
//
// This returns `true` if the cursor successfully moved, and returns `false` if
// there was no previous sibling node.
//
// Note, that this function may be slower than
// [`ts_tree_cursor_goto_next_sibling`] due to how node positions are stored. In
// the worst case, this will need to iterate through all the children upto the
// previous sibling node to recalculate its position.
func (c *TreeCursor) GotoPreviousSibling() bool {
	return bool(C.ts_tree_cursor_goto_previous_sibling(c.c))
}

// GoToFirstChild moves the cursor to the first child of its current node.
//
// This returns `true` if the cursor successfully moved, and returns `false`
// if there were no children.
func (c *TreeCursor) GoToFirstChild() bool {
	return bool(C.ts_tree_cursor_goto_first_child(c.c))
}

// GotoLastChild moves the cursor to the last child of its current node.
//
// This returns `true` if the cursor successfully moved, and returns `false` if
// there were no children.
//
// Note that this function may be slower than [`ts_tree_cursor_goto_first_child`]
// because it needs to iterate through all the children to compute the child's
// position.
func (c *TreeCursor) GotoLastChild() bool {
	return bool(C.ts_tree_cursor_goto_last_child(c.c))
}

// GotoDescendant moves the cursor to the node that is the nth descendant of
// the original node that the cursor was constructed with, where
// zero represents the original node itself.
func (c *TreeCursor) GotoDescendant(goalDescendantIndex uint32) {
	C.ts_tree_cursor_goto_descendant(c.c, C.uint32_t(goalDescendantIndex))
}

// CurrentDescendantIndex returns the index of the cursor's current node out of all of the
// descendants of the original node that the cursor was constructed with.
func (c *TreeCursor) CurrentDescendantIndex() uint32 {
	return uint32(C.ts_tree_cursor_current_descendant_index(c.c))
}

// CurrentDepth returns the depth of the cursor's current node relative to the
// original node that the cursor was constructed with.
func (c *TreeCursor) CurrentDepth() uint32 {
	return uint32(C.ts_tree_cursor_current_depth(c.c))
}

// GoToFirstChildForByte moves the cursor to the first child of its current node
// that extends beyond the given byte offset.
//
// This returns the index of the child node if one was found, and returns -1
// if no such child was found.
func (c *TreeCursor) GoToFirstChildForByte(b uint32) int64 {
	return int64(C.ts_tree_cursor_goto_first_child_for_byte(c.c, C.uint32_t(b)))
}

// GoToFirstChildForPoint moves the cursor to the first child of its current node
// that extends beyond the given point.
//
// This returns the index of the child node if one was found, and returns -1
// if no such child was found.
func (c *TreeCursor) GoToFirstChildForPoint(p Point) int64 {
	return int64(C.ts_tree_cursor_goto_first_child_for_point(c.c, C.TSPoint{
		row: C.uint32_t(p.Row), column: C.uint32_t(p.Column),
	}))
}

// Copy returns a copy of the tree cursor.
func (c *TreeCursor) Copy() *TreeCursor {
	cc := C.ts_tree_cursor_copy(c.c)
	return &TreeCursor{c: &cc, t: c.t.Copy()}
}
