package sitter

// #include "sitter.h"
import "C"

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"unsafe"
)

// BaseTree is needed as we use cache for nodes on normal tree object.
// It prevent run of SetFinalizer as it introduces cycle we can workaround it using
// separate object for details see: https://github.com/golang/go/issues/7358#issuecomment-66091558
type BaseTree struct {
	c    *C.TSTree
	once sync.Once
}

// Tree represents the syntax tree of an entire source code file
// Note: Tree instances are not thread safe;
// you must copy a tree if you want to use it on multiple threads simultaneously.
type Tree struct {
	*BaseTree

	// p is a pointer to a Parser that produced the Tree. Only used to keep Parser alive.
	// Otherwise Parser may be GC'ed (and deleted by the finalizer) while some Tree
	// objects are still in use.
	p *Parser

	cache map[uintptr]*Node
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

// InputEdit represents one edit in the input.
type InputEdit struct {
	StartIndex  uint32
	OldEndIndex uint32
	NewEndIndex uint32
	StartPoint  Point
	OldEndPoint Point
	NewEndPoint Point
}

func (i InputEdit) c() *C.TSInputEdit {
	return &C.TSInputEdit{
		start_byte:    C.uint(i.StartIndex),
		old_end_byte:  C.uint(i.OldEndIndex),
		new_end_byte:  C.uint(i.NewEndIndex),
		start_point:   i.StartPoint.c(),
		old_end_point: i.OldEndPoint.c(),
		new_end_point: i.NewEndPoint.c(),
	}
}

func (p Point) c() C.TSPoint {
	return C.TSPoint{row: C.uint32_t(p.Row), column: C.uint32_t(p.Column)}
}

func mkPoint(p C.TSPoint) Point {
	return Point{Row: uint32(p.row), Column: uint32(p.column)}
}

func (r Range) c() C.TSRange {
	return C.TSRange{
		start_point: r.StartPoint.c(),
		end_point:   r.EndPoint.c(),
		start_byte:  C.uint32_t(r.StartByte),
		end_byte:    C.uint32_t(r.EndByte),
	}
}

func mkRange(r C.TSRange) Range {
	return Range{
		StartPoint: mkPoint(r.start_point),
		EndPoint:   mkPoint(r.end_point),
		StartByte:  uint32(r.start_byte),
		EndByte:    uint32(r.end_byte),
	}
}

func mkRanges(p *C.TSRange, count C.uint32_t) (out []Range) {
	out = make([]Range, count)

	for i, r := range unsafe.Slice(p, int(count)) {
		out[i] = mkRange(r)
	}

	return
}

// newTree creates a new tree object from a C pointer. The function will set a finalizer for the object,
// thus no free is needed for it.
func (p *Parser) newTree(c *C.TSTree) *Tree {
	base := &BaseTree{c: c}

	runtime.SetFinalizer(base, (*BaseTree).close)

	return &Tree{p: p, BaseTree: base, cache: map[uintptr]*Node{}}
}

// Copy creates a shallow copy of the syntax tree. This is very fast.
//
// You need to copy a syntax tree in order to use it on more than one thread at
// a time, as syntax trees are not thread safe.
func (t *Tree) Copy() *Tree {
	return t.p.newTree(C.ts_tree_copy(t.c))
}

// close should be called to ensure that all the memory used by the tree is freed.
//
// As the constructor in go-tree-sitter would set this func call through runtime.SetFinalizer,
// parser.close() will be called by Go's garbage collector and users need not call this manually.
func (t *BaseTree) close() {
	t.once.Do(func() { C.ts_tree_delete(t.c) })
}

// RootNode returns root node of the syntax tree.
func (t *Tree) RootNode() *Node {
	nn := C.ts_tree_root_node(t.c)
	return t.cachedNode(nn)
}

// RootNodeWithOffset returns the root node of the syntax tree, but with its position
// shifted forward by the given offset.
func (t *Tree) RootNodeWithOffset(ofs uint32, extent Point) *Node {
	nn := C.ts_tree_root_node_with_offset(t.c, C.uint32_t(ofs), extent.c())
	return t.cachedNode(nn)
}

// Language returns the language that was used to parse the syntax tree.
func (t *Tree) Language() *Language {
	return newLanguage(C.ts_tree_language(t.c))
}

// IncludedRanges returns the array of included ranges that was used to parse the syntax tree.
//
// The returned pointer must be freed by the caller.
func (t *Tree) IncludedRanges() []Range {
	count := C.uint(0)

	p := C.ts_tree_included_ranges(t.c, &count)
	defer freeTSRangeArray(p, count)

	return mkRanges(p, count)
}

// Edit the syntax tree to keep it in sync with source code that has been edited.
//
// You MUST describe the edit both in terms of byte offsets and in terms of
// (row, column) coordinates.
func (t *Tree) Edit(i InputEdit) {
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
func (t *Tree) GetChangedRanges(other *Tree) []Range {
	count := C.uint(0)

	p := C.ts_tree_get_changed_ranges(t.c, other.c, &count)
	defer freeTSRangeArray(p, count)

	return mkRanges(p, count)
}

// PrintDotGraph writes a DOT graph describing the syntax tree to the given file.
func (t *Tree) PrintDotGraph(name string) (err error) {
	f, err := os.Create(name)
	if err != nil {
		return
	}

	C.ts_tree_print_dot_graph(t.c, C.int(f.Fd()))

	if err = f.Close(); err != nil {
		err = fmt.Errorf("cannot save dot file: %w", err)
	}

	return
}

func (t *Tree) cachedNode(ptr C.TSNode) (n *Node) {
	if ptr.id == nil {
		return
	}

	var ok bool

	p := uintptr(ptr.id)
	if n, ok = t.cache[p]; ok {
		return
	}

	n = &Node{c: ptr, t: t}
	t.cache[p] = n

	return
}

func freeTSRangeArray(p *C.struct_TSRange, count C.uint) {
	pp := unsafe.Pointer(p)

	for ; count > 0; count-- {
		C.free(pp)
		pp = unsafe.Add(pp, C.sizeof_struct_TSRange)
	}
}
