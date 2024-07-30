package sitter

// #include "sitter.h"
import "C" //nolint:gocritic // ok

import (
	"runtime"
	"unsafe" //nolint:gocritic // ok
)

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

	return &Tree{p: p, BaseTree: base, cache: map[C.TSNode]*Node{}}
}

// Copy creates a shallow copy of the syntax tree. This is very fast.
//
// You need to copy a syntax tree in order to use it on more than one thread at
// a time, as syntax trees are not thread safe.
func (t *Tree) Copy() *Tree {
	return t.p.newTree(C.ts_tree_copy(t.c))
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

// RootNode returns root node of the syntax tree.
func (t *Tree) RootNode() *Node {
	ptr := C.ts_tree_root_node(t.c)
	return t.cachedNode(ptr)
}

/**
 * Get the root node of the syntax tree, but with its position
 * shifted forward by the given offset.
TSNode ts_tree_root_node_with_offset(
  const TSTree *self,
  uint32_t offset_bytes,
  TSPoint offset_extent
);
*/

// Language returns the language that was used to parse the syntax tree.
func (t *Tree) Language() Language {
	return Language{ptr: unsafe.Pointer(C.ts_tree_language(t.c))}
}

/**
 * Get the array of included ranges that was used to parse the syntax tree.
 *
 * The returned pointer must be freed by the caller.
TSRange *ts_tree_included_ranges(const TSTree *self, uint32_t *length);
*/

// Edit the syntax tree to keep it in sync with source code that has been edited.
//
// You MUST describe the edit both in terms of byte offsets and in terms of
// (row, column) coordinates.
func (t *Tree) Edit(i EditInput) {
	C.ts_tree_edit(t.c, i.c())
}

// GetChangedRanges compares an old edited syntax tree to a new syntax tree
// representing the same  document, returning an array of ranges whose
// syntactic structure has changed.
//
// For this to work correctly, the old syntax tree must have been edited such
// that its ranges match up to the new tree. Generally, you'll want to call
// this function right after calling one of the [`ts_parser_parse`] functions.
// You need to pass the old tree that was passed to parse, as well as the new
// tree that was returned from that function.
//
// The returned array is allocated using `malloc` and the caller is responsible
// for freeing it using `free`. The length of the array will be written to the
// given `length` pointer.
// TODO: Add unit tests.
func (t *Tree) GetChangedRanges(other *Tree, length uint32) *Range {
	l := C.uint32_t(length)
	r := C.ts_tree_get_changed_ranges(t.c, other.c, &l)

	return &Range{
		StartPoint: Point{Row: uint32(r.start_point.row), Column: uint32(r.start_point.column)},
		EndPoint:   Point{Row: uint32(r.end_point.row), Column: uint32(r.end_point.column)},
		StartByte:  uint32(r.start_byte),
		EndByte:    uint32(r.end_byte),
	}
}

/**
 * Write a DOT graph describing the syntax tree to the given file.
void ts_tree_print_dot_graph(const TSTree *self, int file_descriptor);
*/

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
