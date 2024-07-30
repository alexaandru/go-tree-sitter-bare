package sitter

// #include "sitter.h"
import "C"      //nolint:gocritic // ok
import "unsafe" //nolint:gocritic // ok

// Node represents a single node in the syntax tree
// It tracks its start and end positions in the source code,
// as well as its relation to other nodes like its parent, siblings and children.
type Node struct {
	//	TODO: uint32_t context[4];
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

// Type returns the node's type as a string.
func (n Node) Type() string {
	return C.GoString(C.ts_node_type(n.c))
}

// Symbol returns the node's type as a Symbol.
func (n Node) Symbol() Symbol {
	return C.ts_node_symbol(n.c)
}

// Language returns the node's language.
func (n Node) Language() Language {
	return Language{ptr: unsafe.Pointer(C.ts_node_language(n.c))}
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
func (n Node) String() string {
	p := C.ts_node_string(n.c)
	defer C.free(unsafe.Pointer(p))

	return C.GoString(p)
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

// HasChanges checks if a syntax node has been edited.
func (n Node) HasChanges() bool {
	return bool(C.ts_node_has_changes(n.c))
}

// HasError check if the node is a syntax error or contains any syntax errors.
func (n Node) HasError() bool {
	return bool(C.ts_node_has_error(n.c))
}

// IsError checks if the node is a syntax error.
// Syntax errors represent parts of the code that could not be incorporated into a valid syntax tree.
func (n Node) IsError() bool {
	return bool(C.ts_node_is_error(n.c))
}

/**
 * Get this node's parse state.
/
TSStateId ts_node_parse_state(TSNode self);

/**
 * Get the parse state after this node.
/
TSStateId ts_node_next_parse_state(TSNode self);
*/

// Parent returns the node's immediate parent.
func (n Node) Parent() *Node {
	nn := C.ts_node_parent(n.c)
	return n.t.cachedNode(nn)
}

// ChildContainingDescendant returns the node's child that contains `descendant`.
func (n Node) ChildContainingDescendant(d *Node) *Node {
	nn := C.ts_node_child_containing_descendant(n.c, d.c)
	return n.t.cachedNode(nn)
}

// Child returns the node's child at the given index, where zero represents the first child.
func (n Node) Child(idx uint32) *Node {
	nn := C.ts_node_child(n.c, C.uint32_t(idx))
	return n.t.cachedNode(nn)
}

// FieldNameForChild returns the field name of the child at the given index, or "" if not named.
func (n Node) FieldNameForChild(idx int) string {
	return C.GoString(C.ts_node_field_name_for_child(n.c, C.uint32_t(idx)))
}

// ChildCount returns the node's number of children.
func (n Node) ChildCount() uint32 {
	return uint32(C.ts_node_child_count(n.c))
}

// NamedChild returns the node's *named* child at the given index.
func (n Node) NamedChild(idx uint32) *Node {
	nn := C.ts_node_named_child(n.c, C.uint32_t(idx))
	return n.t.cachedNode(nn)
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

/**
 * Get the node's child with the given numerical field id.
 *
 * You can convert a field name to an id using the
 * [`ts_language_field_id_for_name`] function.
/
TSNode ts_node_child_by_field_id(TSNode self, TSFieldId field_id);
*/

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

/**
 * Get the node's first child that extends beyond the given byte offset.
 /
TSNode ts_node_first_child_for_byte(TSNode self, uint32_t byte);

/**
 * Get the node's first named child that extends beyond the given byte offset.
 /
TSNode ts_node_first_named_child_for_byte(TSNode self, uint32_t byte);

/**
 * Get the node's number of descendants, including one for the node itself.
 /
uint32_t ts_node_descendant_count(TSNode self);

/**
 * Get the smallest node within this node that spans the given range of bytes
 * or (row, column) positions.
 /
TSNode ts_node_descendant_for_byte_range(TSNode self, uint32_t start, uint32_t end);
TSNode ts_node_descendant_for_point_range(TSNode self, TSPoint start, TSPoint end);
*/

// NamedDescendantForByteRange returns the smallest named node within this node
// that spans the given range of bytes.
func (n Node) NamedDescendantForByteRange(start, end uint32) *Node {
	nn := C.ts_node_named_descendant_for_byte_range(n.c, C.uint32_t(start), C.uint32_t(end))
	return n.t.cachedNode(nn)
}

// NamedDescendantForPointRange returns the smallest named node within this node
// that spans the given range of row/column positions.
func (n Node) NamedDescendantForPointRange(start, end Point) *Node {
	cStart := C.TSPoint{row: C.uint32_t(start.Row), column: C.uint32_t(start.Column)}
	cEnd := C.TSPoint{row: C.uint32_t(end.Row), column: C.uint32_t(end.Column)}
	nn := C.ts_node_named_descendant_for_point_range(n.c, cStart, cEnd)

	return n.t.cachedNode(nn)
}

// Edit the node to keep it in-sync with source code that has been edited.
//
// This function is only rarely needed. When you edit a syntax tree with the
// `ts_tree_edit` function, all of the nodes that you retrieve from the tree
// afterward will already reflect the edit. You only need to use `ts_node_edit`
// when you have a `TSNode` instance that you want to keep and continue to use
// after an edit.
func (n Node) Edit(i EditInput) {
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
