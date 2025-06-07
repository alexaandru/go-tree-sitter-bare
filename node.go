package sitter

// #include "sitter.h"
import "C"

import "unsafe"

// Node represents a single node in the syntax tree
// It tracks its start and end positions in the source code,
// as well as its relation to other nodes like its parent, siblings and children.
type Node struct {
	c C.TSNode
}

// Symbol indicates the symbol.
type Symbol = C.TSSymbol

// SymbolType indicates the type of symbol.
type SymbolType uint32

// Possible symbol types.
const (
	SymbolTypeRegular   SymbolType = C.TSSymbolTypeRegular
	SymbolTypeAnonymous SymbolType = C.TSSymbolTypeAnonymous
	SymbolTypeSupertype SymbolType = C.TSSymbolTypeSupertype
	SymbolTypeAuxiliary SymbolType = C.TSSymbolTypeAuxiliary
)

func newNode(ptr C.TSNode) (n Node) {
	if ptr.id == nil {
		return
	}

	return Node{c: ptr}
}

// Type returns the node's type.
func (n Node) Type() string {
	return C.GoString(C.ts_node_type(n.c))
}

// Symbol returns the node's type.
func (n Node) Symbol() Symbol {
	return C.ts_node_symbol(n.c)
}

// Language returns the node's language.
func (n Node) Language() *Language {
	return NewLanguage(unsafe.Pointer(C.ts_node_language(n.c)))
}

// GrammarType returns the node's type as it appears in the grammar,
// ignoring aliases.
func (n Node) GrammarType() string {
	return C.GoString(C.ts_node_grammar_type(n.c))
}

// GrammarSymbol returns the node's symbol as it appears in the grammar,
// ignoring aliases.
// This should be used in `ts_language_next_state` instead of `ts_node_symbol`.
func (n Node) GrammarSymbol() Symbol {
	return Symbol(C.ts_node_grammar_symbol(n.c)) //nolint:unconvert // we need the methods on the aliased type
}

// StartByte returns the node's start byte.
func (n Node) StartByte() uint {
	return uint(C.ts_node_start_byte(n.c))
}

// StartPoint returns the node's start position in terms of rows and columns.
func (n Node) StartPoint() Point {
	return mkPoint(C.ts_node_start_point(n.c))
}

// EndByte returns the node's end byte.
func (n Node) EndByte() uint {
	return uint(C.ts_node_end_byte(n.c))
}

// EndPoint returns the node's end position in terms of rows and columns.
func (n Node) EndPoint() Point {
	return mkPoint(C.ts_node_end_point(n.c))
}

// String returns an S-expression representing the node as a string.
//
// This string is allocated with `malloc` and the caller is responsible for
// freeing it using `free`.
func (n Node) String() string {
	p := C.ts_node_string(n.c)

	defer C.free(unsafe.Pointer(p))

	return C.GoString(p)
}

// IsNull checks if the node is null.
//
// Functions like `ts_node_child` and `ts_node_next_sibling` will return a null
// node to indicate that no such node was found.
func (n Node) IsNull() bool {
	return bool(C.ts_node_is_null(n.c))
}

// IsNamed checks if the node is *named*.
//
// Named nodes correspond to named rules in the grammar,
// whereas *anonymous* nodes correspond to string literals in the grammar.
func (n Node) IsNamed() bool {
	return bool(C.ts_node_is_named(n.c))
}

// IsMissing checks if the node is *missing*.
//
// Missing nodes are inserted by the parser in order to recover from certain
// kinds of syntax errors.
func (n Node) IsMissing() bool {
	return bool(C.ts_node_is_missing(n.c))
}

// IsExtra checks if the node is *extra*.
//
// Extra nodes represent things like comments, which are not required the grammar,
// but can appear anywhere.
func (n Node) IsExtra() bool {
	return bool(C.ts_node_is_extra(n.c))
}

// HasChanges checks if a syntax node has been edited.
func (n Node) HasChanges() bool {
	return bool(C.ts_node_has_changes(n.c))
}

// HasError check if the node is a syntax error or contains any syntax errors.
func (n Node) HasError() bool {
	return bool(C.ts_node_has_error(n.c))
}

// IsError checks if the node is a syntax error.
//
// Syntax errors represent parts of the code that could not be incorporated
// into a valid syntax tree.
func (n Node) IsError() bool {
	return bool(C.ts_node_is_error(n.c))
}

// ParseState returns this node's parse state.
func (n Node) ParseState() StateID {
	return C.ts_node_parse_state(n.c)
}

// NextParseState returns the parse state after this node.
func (n Node) NextParseState() StateID {
	return C.ts_node_next_parse_state(n.c)
}

// Parent returns the node's immediate parent.
//
// Prefer `ts_node_child_with_descendant` for
// iterating over the node's ancestors.
func (n Node) Parent() Node {
	return newNode(C.ts_node_parent(n.c))
}

// ChildWithDescendant returns the node that contains `descendant`.
//
// NOTE: that this can return `descendant` itself.
func (n Node) ChildWithDescendant(d Node) Node {
	return newNode(C.ts_node_child_with_descendant(n.c, d.c))
}

// Child returns the node's child at the given index, where zero represents the
// first child.
func (n Node) Child(idx uint32) Node {
	return newNode(C.ts_node_child(n.c, C.uint(idx)))
}

// FieldNameForChild returns the field name of the child at the given index,
// or "" if not named.
func (n Node) FieldNameForChild(idx int) string {
	return C.GoString(C.ts_node_field_name_for_child(n.c, C.uint(idx)))
}

// FieldNameForNamedChild returns the field name for node's named child at the given index, where zero
// represents the first named child. Returns NULL, if no field is found.
func (n Node) FieldNameForNamedChild(idx uint32) string {
	return C.GoString(C.ts_node_field_name_for_named_child(n.c, C.uint(idx)))
}

// ChildCount returns the node's number of children.
func (n Node) ChildCount() uint32 {
	return uint32(C.ts_node_child_count(n.c))
}

// NamedChild returns the node's *named* child at the given index.
//
// See also `ts_node_is_named`.
func (n Node) NamedChild(idx uint32) Node {
	return newNode(C.ts_node_named_child(n.c, C.uint(idx)))
}

// NamedChildCount returns the node's number of *named* children.
//
// See also `ts_node_is_named`.
func (n Node) NamedChildCount() uint32 {
	return uint32(C.ts_node_named_child_count(n.c))
}

// ChildByFieldName returns the node's child with the given field name.
func (n Node) ChildByFieldName(name string) Node {
	str := C.CString(name)

	defer C.free(unsafe.Pointer(str))

	return newNode(C.ts_node_child_by_field_name(n.c, str, C.uint(len(name))))
}

// ChildByFieldID returns the node's child with the given numerical field id.
//
// You can convert a field name to an id using the
// `ts_language_field_id_for_name` function.
func (n Node) ChildByFieldID(id FieldID) Node {
	return newNode(C.ts_node_child_by_field_id(n.c, id))
}

// NextSibling returns the node's next sibling.
func (n Node) NextSibling() Node {
	return newNode(C.ts_node_next_sibling(n.c))
}

// PrevSibling returns the node's previous sibling.
func (n Node) PrevSibling() Node {
	return newNode(C.ts_node_prev_sibling(n.c))
}

// NextNamedSibling returns the node's next *named* sibling.
func (n Node) NextNamedSibling() Node {
	return newNode(C.ts_node_next_named_sibling(n.c))
}

// PrevNamedSibling returns the node's previous *named* sibling.
func (n Node) PrevNamedSibling() Node {
	return newNode(C.ts_node_prev_named_sibling(n.c))
}

// FirstChildForByte returns the node's first child that contains or starts after
// the given byte offset.
func (n Node) FirstChildForByte(ofs uint32) Node {
	return newNode(C.ts_node_first_child_for_byte(n.c, C.uint(ofs)))
}

// FirstNamedChildForByte returns the node's first named child that contains or starts
// after the given byte offset.
func (n Node) FirstNamedChildForByte(ofs uint32) Node {
	return newNode(C.ts_node_first_named_child_for_byte(n.c, C.uint(ofs)))
}

// DescendantCount returns the node's number of descendants, including one
// for the node itself.
func (n Node) DescendantCount() uint32 {
	return uint32(C.ts_node_descendant_count(n.c))
}

// DescendantForByteRange returns the smallest node within this node that spans
// the given range of bytes.
func (n Node) DescendantForByteRange(start, end uint32) Node {
	return newNode(C.ts_node_descendant_for_byte_range(n.c, C.uint(start), C.uint(end)))
}

// DescendantForPointRange returns the smallest node within this node that spans
// the given range of {row, column} positions.
func (n Node) DescendantForPointRange(start, end Point) Node {
	return newNode(C.ts_node_descendant_for_point_range(n.c, start.c(), end.c()))
}

// NamedDescendantForByteRange returns the smallest named node within this node
// that spans the given range of bytes.
func (n Node) NamedDescendantForByteRange(start, end uint32) Node {
	return newNode(C.ts_node_named_descendant_for_byte_range(n.c, C.uint(start), C.uint(end)))
}

// NamedDescendantForPointRange returns the smallest named node within this node
// that spans the given range of row/column positions.
func (n Node) NamedDescendantForPointRange(start, end Point) Node {
	return newNode(C.ts_node_named_descendant_for_point_range(n.c, start.c(), end.c()))
}

// Edit the node to keep it in-sync with source code that has been edited.
//
// This function is only rarely needed. When you edit a syntax tree with the
// `ts_tree_edit` function, all of the nodes that you retrieve from the tree
// afterward will already reflect the edit. You only need to use `ts_node_edit`
// when you have a `TSNode` instance that you want to keep and continue to use
// after an edit.
func (n Node) Edit(i InputEdit) {
	C.ts_node_edit(&n.c, i.c()) //nolint:gocritic // ok
}

// Equal checks if two nodes are identical.
func (n Node) Equal(other Node) bool {
	return bool(C.ts_node_eq(n.c, other.c))
}

// Non API.

// Range returns the node range.
func (n Node) Range() Range {
	return Range{
		StartByte: n.StartByte(), EndByte: n.EndByte(),
		StartPoint: n.StartPoint(), EndPoint: n.EndPoint(),
	}
}

// Content returns node's source code from input as a string.
func (n Node) Content(input []byte) string {
	return string(input[n.StartByte():n.EndByte()])
}
