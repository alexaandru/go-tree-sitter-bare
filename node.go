package sitter

// #include "bindings.h"
import "C" //nolint:gocritic // ok

import (
	"math"
	"unsafe" //nolint:gocritic // ok
)

// Node represents a single node in the syntax tree
// It tracks its start and end positions in the source code,
// as well as its relation to other nodes like its parent, siblings and children.
type Node struct {
	c C.TSNode
	t *Tree // keep pointer on tree because node is valid only as long as tree is
}

// Symbol indicates the type of symbol.
type Symbol = C.TSSymbol

// SymbolType is the corresponding Go type for Symbol.
type SymbolType int

// Possible symbol types.
const (
	SymbolTypeRegular SymbolType = iota
	SymbolTypeAnonymous
	SymbolTypeAuxiliary
)

var symbolTypeNames = []string{
	"Regular",
	"Anonymous",
	"Auxiliary",
}

func (t SymbolType) String() string {
	return symbolTypeNames[t]
}

// ID returns the node ID.
func (n Node) ID() uintptr {
	return uintptr(n.c.id)
}

// StartByte returns the node's start byte.
func (n Node) StartByte() uint32 {
	return uint32(C.ts_node_start_byte(n.c))
}

// EndByte returns the node's end byte.
func (n Node) EndByte() uint32 {
	return uint32(C.ts_node_end_byte(n.c))
}

// StartPoint returns the node's start position in terms of rows and columns.
func (n Node) StartPoint() Point {
	p := C.ts_node_start_point(n.c)
	return Point{Row: uint32(p.row), Column: uint32(p.column)}
}

// EndPoint returns the node's end position in terms of rows and columns.
func (n Node) EndPoint() Point {
	p := C.ts_node_end_point(n.c)
	return Point{Row: uint32(p.row), Column: uint32(p.column)}
}

// Range returns the node range.
func (n Node) Range() Range {
	return Range{
		StartByte:  n.StartByte(),
		EndByte:    n.EndByte(),
		StartPoint: n.StartPoint(),
		EndPoint:   n.EndPoint(),
	}
}

// Symbol returns the node's type as a Symbol.
func (n Node) Symbol() Symbol {
	return C.ts_node_symbol(n.c)
}

// Type returns the node's type as a string.
func (n Node) Type() string {
	return C.GoString(C.ts_node_type(n.c))
}

// String returns an S-expression representing the node as a string.
func (n Node) String() string {
	ptr := C.ts_node_string(n.c)

	defer C.free(unsafe.Pointer(ptr))

	return C.GoString(ptr)
}

// Equal checks if two nodes are identical.
func (n Node) Equal(other *Node) bool {
	return bool(C.ts_node_eq(n.c, other.c))
}

// IsNull checks if the node is null.
func (n Node) IsNull() bool {
	return bool(C.ts_node_is_null(n.c))
}

// IsNamed checks if the node is *named*.
// Named nodes correspond to named rules in the grammar,
// whereas *anonymous* nodes correspond to string literals in the grammar.
func (n Node) IsNamed() bool {
	return bool(C.ts_node_is_named(n.c))
}

// IsMissing checks if the node is *missing*.
// Missing nodes are inserted by the parser in order to recover from certain kinds of syntax errors.
func (n Node) IsMissing() bool {
	return bool(C.ts_node_is_missing(n.c))
}

// IsExtra checks if the node is *extra*.
// Extra nodes represent things like comments, which are not required the grammar, but can appear anywhere.
func (n Node) IsExtra() bool {
	return bool(C.ts_node_is_extra(n.c))
}

// IsError checks if the node is a syntax error.
// Syntax errors represent parts of the code that could not be incorporated into a valid syntax tree.
func (n Node) IsError() bool {
	return n.Symbol() == math.MaxUint16
}

// HasChanges checks if a syntax node has been edited.
func (n Node) HasChanges() bool {
	return bool(C.ts_node_has_changes(n.c))
}

// HasError check if the node is a syntax error or contains any syntax errors.
func (n Node) HasError() bool {
	return bool(C.ts_node_has_error(n.c))
}

// Parent returns the node's immediate parent.
func (n Node) Parent() *Node {
	nn := C.ts_node_parent(n.c)
	return n.t.cachedNode(nn)
}

// Child returns the node's child at the given index, where zero represents the first child.
func (n Node) Child(idx int) *Node {
	nn := C.ts_node_child(n.c, C.uint32_t(idx))
	return n.t.cachedNode(nn)
}

// NamedChild returns the node's *named* child at the given index.
func (n Node) NamedChild(idx int) *Node {
	nn := C.ts_node_named_child(n.c, C.uint32_t(idx))
	return n.t.cachedNode(nn)
}

// ChildCount returns the node's number of children.
func (n Node) ChildCount() uint32 {
	return uint32(C.ts_node_child_count(n.c))
}

// NamedChildCount returns the node's number of *named* children.
func (n Node) NamedChildCount() uint32 {
	return uint32(C.ts_node_named_child_count(n.c))
}

// ChildByFieldName returns the node's child with the given field name.
func (n Node) ChildByFieldName(name string) *Node {
	str := C.CString(name)
	defer C.free(unsafe.Pointer(str))

	nn := C.ts_node_child_by_field_name(n.c, str, C.uint32_t(len(name)))

	return n.t.cachedNode(nn)
}

// FieldNameForChild returns the field name of the child at the given index, or "" if not named.
func (n Node) FieldNameForChild(idx int) string {
	return C.GoString(C.ts_node_field_name_for_child(n.c, C.uint32_t(idx)))
}

// NextSibling returns the node's next sibling.
func (n Node) NextSibling() *Node {
	nn := C.ts_node_next_sibling(n.c)
	return n.t.cachedNode(nn)
}

// NextNamedSibling returns the node's next *named* sibling.
func (n Node) NextNamedSibling() *Node {
	nn := C.ts_node_next_named_sibling(n.c)
	return n.t.cachedNode(nn)
}

// PrevSibling returns the node's previous sibling.
func (n Node) PrevSibling() *Node {
	nn := C.ts_node_prev_sibling(n.c)
	return n.t.cachedNode(nn)
}

// PrevNamedSibling returns the node's previous *named* sibling.
func (n Node) PrevNamedSibling() *Node {
	nn := C.ts_node_prev_named_sibling(n.c)
	return n.t.cachedNode(nn)
}

// ChildContainingDescendant returns the node's child that contains `descendant`.
func (n Node) ChildContainingDescendant(d *Node) *Node {
	nn := C.ts_node_child_containing_descendant(n.c, d.c)
	return n.t.cachedNode(nn)
}

// Edit the node to keep it in-sync with source code that has been edited.
func (n Node) Edit(i EditInput) {
	C.ts_node_edit(&n.c, i.c()) //nolint:gocritic // ok
}

// Content returns node's source code from input as a string
func (n Node) Content(input []byte) string {
	return string(input[n.StartByte():n.EndByte()])
}

// NamedDescendantForByteRange returns the smallest named node within this node
// that spans the given range of bytes.
func (n Node) NamedDescendantForByteRange(start, end uint32) *Node {
	nn := C.ts_node_named_descendant_for_byte_range(n.c, C.uint32_t(start), C.uint32_t(end))
	return n.t.cachedNode(nn)
}

// NamedDescendantForPointRange returns the smallest named node within this node
// that spans the given range of row/column positions.
func (n Node) NamedDescendantForPointRange(start, end Point) *Node {
	cStartPoint := C.TSPoint{
		row:    C.uint32_t(start.Row),
		column: C.uint32_t(start.Column),
	}
	cEndPoint := C.TSPoint{
		row:    C.uint32_t(end.Row),
		column: C.uint32_t(end.Column),
	}
	nn := C.ts_node_named_descendant_for_point_range(n.c, cStartPoint, cEndPoint)

	return n.t.cachedNode(nn)
}
