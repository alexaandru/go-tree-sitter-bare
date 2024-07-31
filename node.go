package sitter

// #include "sitter.h"
import "C"      //nolint:gocritic // ok
import "unsafe" //nolint:gocritic // ok

// Node represents a single node in the syntax tree
// It tracks its start and end positions in the source code,
// as well as its relation to other nodes like its parent, siblings and children.
type Node struct {
	c       C.TSNode
	t       *Tree     // keep pointer on tree because node is valid only as long as tree is
	context [4]uint32 // TODO: How is this used upstream?
}

// Symbol indicates the type of symbol.
type Symbol = C.TSSymbol

// Possible symbol types.
const (
	SymbolTypeRegular Symbol = iota
	SymbolTypeAnonymous
	SymbolTypeAuxiliary
)

var symbolTypeNames = []string{
	"Regular",
	"Anonymous",
	"Auxiliary",
}

func (t Symbol) String() string {
	return symbolTypeNames[t]
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
	return newLanguage(C.ts_node_language(n.c))
}

// GrammarType returns the node's type as it appears in the grammar,
// ignoring aliases.
func (n Node) GrammarType() string {
	p := C.ts_node_grammar_type(n.c)
	defer C.free(unsafe.Pointer(p))

	return C.GoString(p)
}

// GrammarSymbol returns the node's symbol as it appears in the grammar,
// ignoring aliases.
// This should be used in `ts_language_next_state` instead of `ts_node_symbol`.
func (n Node) GrammarSymbol() Symbol {
	return C.ts_node_grammar_symbol(n.c)
}

// StartByte returns the node's start byte.
func (n Node) StartByte() uint32 {
	return uint32(C.ts_node_start_byte(n.c))
}

// StartPoint returns the node's start position in terms of rows and columns.
func (n Node) StartPoint() Point {
	p := C.ts_node_start_point(n.c)
	return Point{Row: uint32(p.row), Column: uint32(p.column)}
}

// EndByte returns the node's end byte.
func (n Node) EndByte() uint32 {
	return uint32(C.ts_node_end_byte(n.c))
}

// EndPoint returns the node's end position in terms of rows and columns.
func (n Node) EndPoint() Point {
	p := C.ts_node_end_point(n.c)
	return Point{Row: uint32(p.row), Column: uint32(p.column)}
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
// Prefer `ts_node_child_containing_descendant` for
// iterating over the node's ancestors.
func (n Node) Parent() *Node {
	nn := C.ts_node_parent(n.c)
	return n.t.cachedNode(nn)
}

// ChildContainingDescendant returns the node's child that contains `descendant`.
func (n Node) ChildContainingDescendant(d *Node) *Node {
	nn := C.ts_node_child_containing_descendant(n.c, d.c)
	return n.t.cachedNode(nn)
}

// Child returns the node's child at the given index, where zero represents the
// first child.
func (n Node) Child(idx uint32) *Node {
	nn := C.ts_node_child(n.c, C.uint32_t(idx))
	return n.t.cachedNode(nn)
}

// FieldNameForChild returns the field name of the child at the given index,
// or "" if not named.
func (n Node) FieldNameForChild(idx int) string {
	return C.GoString(C.ts_node_field_name_for_child(n.c, C.uint32_t(idx)))
}

// ChildCount returns the node's number of children.
func (n Node) ChildCount() uint32 {
	return uint32(C.ts_node_child_count(n.c))
}

// NamedChild returns the node's *named* child at the given index.
//
// See also `ts_node_is_named`.
func (n Node) NamedChild(idx uint32) *Node {
	nn := C.ts_node_named_child(n.c, C.uint32_t(idx))
	return n.t.cachedNode(nn)
}

// NamedChildCount returns the node's number of *named* children.
//
// See also `ts_node_is_named`.
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

// ChildByFieldID returns the node's child with the given numerical field id.
//
// You can convert a field name to an id using the
// `ts_language_field_id_for_name` function.
func (n Node) ChildByFieldID(id FieldID) *Node {
	nn := C.ts_node_child_by_field_id(n.c, id)
	return n.t.cachedNode(nn)
}

// NextSibling returns the node's next sibling.
func (n Node) NextSibling() *Node {
	nn := C.ts_node_next_sibling(n.c)
	return n.t.cachedNode(nn)
}

// PrevSibling returns the node's previous sibling.
func (n Node) PrevSibling() *Node {
	nn := C.ts_node_prev_sibling(n.c)
	return n.t.cachedNode(nn)
}

// NextNamedSibling returns the node's next *named* sibling.
func (n Node) NextNamedSibling() *Node {
	nn := C.ts_node_next_named_sibling(n.c)
	return n.t.cachedNode(nn)
}

// PrevNamedSibling returns the node's previous *named* sibling.
func (n Node) PrevNamedSibling() *Node {
	nn := C.ts_node_prev_named_sibling(n.c)
	return n.t.cachedNode(nn)
}

// FirstChildForByte returns the node's first child that extends beyond the
// given byte offset.
func (n Node) FirstChildForByte(ofs uint32) *Node {
	nn := C.ts_node_first_child_for_byte(n.c, C.uint32_t(ofs))
	return n.t.cachedNode(nn)
}

// FirstNamedChildForByte returns the node's first named child that extends
// beyond the given byte offset.
func (n Node) FirstNamedChildForByte(ofs uint32) *Node {
	nn := C.ts_node_first_named_child_for_byte(n.c, C.uint32_t(ofs))
	return n.t.cachedNode(nn)
}

// DescendantCount returns the node's number of descendants, including one
// for the node itself.
func (n Node) DescendantCount() uint32 {
	return uint32(C.ts_node_descendant_count(n.c))
}

// DescendantForByteRange returns the smallest node within this node that spans
// the given range of bytes.
func (n Node) DescendantForByteRange(start, end uint32) *Node {
	nn := C.ts_node_descendant_for_byte_range(n.c, C.uint32_t(start), C.uint32_t(end))
	return n.t.cachedNode(nn)
}

// DescendantForPointRange returns the smallest node within this node that spans
// the given range of {row, column} positions.
func (n Node) DescendantForPointRange(start, end Point) *Node {
	nn := C.ts_node_descendant_for_point_range(n.c, mkCPoint(start), mkCPoint(end))
	return n.t.cachedNode(nn)
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
	nn := C.ts_node_named_descendant_for_point_range(n.c, mkCPoint(start), mkCPoint(end))
	return n.t.cachedNode(nn)
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
func (n Node) Equal(other *Node) bool {
	return bool(C.ts_node_eq(n.c, other.c))
}

// Non API.

// ID returns the node ID.
func (n Node) ID() uintptr {
	return uintptr(n.c.id)
}

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
