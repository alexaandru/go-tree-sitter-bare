package sitter

// #include "sitter.h"
import "C"

import (
	"fmt"
	"sync"
	"unsafe"
)

// Language defines how to parse a particular programming language.
type Language struct {
	ptr  unsafe.Pointer
	once sync.Once
}

// LanguageError represents an error  that occurred when trying to assign
// an incompatible [TSLanguage] to a [TSParser].
type LanguageError int

// StateID is used for parser state ID.
type StateID = C.TSStateId

// FieldID is used for parser field ID.
type FieldID = C.TSFieldId

// Copy returns another reference to the language.
func (l *Language) Copy() *Language {
	return NewLanguage(unsafe.Pointer(C.ts_language_copy(l.c())))
}

// Delete frees any dynamically-allocated resources for this language, if
// this is the last reference.
func (l *Language) Delete() {
	l.once.Do(func() {
		C.ts_language_delete(l.c())

		l.ptr = nil
		l = nil
	})
}

// SymbolCount returns the number of distinct field names in the language.
func (l *Language) SymbolCount() uint32 {
	return uint32(C.ts_language_symbol_count(l.c()))
}

// StateCount returns the number of valid states in the language.
func (l *Language) StateCount() uint32 {
	return uint32(C.ts_language_state_count(l.c()))
}

// SymbolName returns a node type string for the given Symbol.
func (l *Language) SymbolName(s Symbol) string {
	return C.GoString(C.ts_language_symbol_name(l.c(), s))
}

// SymbolID returns the numerical id for the given node type string.
func (l *Language) SymbolID(name string, isNamed bool) Symbol {
	cName := C.CString(name)

	defer C.free(unsafe.Pointer(cName))

	return C.ts_language_symbol_for_name(l.c(), cName, C.uint(len(name)), C._Bool(isNamed))
}

// FieldCount returns the number of distinct field names in the language.
func (l *Language) FieldCount() uint32 {
	return uint32(C.ts_language_field_count(l.c()))
}

// FieldName returns the field name string for the given numerical id.
func (l *Language) FieldName(idx int) string {
	return C.GoString(C.ts_language_field_name_for_id(l.c(), C.ushort(idx)))
}

// FieldID returns the numerical id for the given field name string.
func (l *Language) FieldID(name string) FieldID {
	cName := C.CString(name)

	defer C.free(unsafe.Pointer(cName))

	return C.ts_language_field_id_for_name(l.c(), cName, C.uint(len(name)))
}

// SymbolType returns named, anonymous, or a hidden type for a Symbol.
func (l *Language) SymbolType(s Symbol) SymbolType {
	return SymbolType(C.ts_language_symbol_type(l.c(), s)) //nolint:unconvert // ok
}

// Version returns the ABI version number for this language. This version number is used
// to ensure that languages were generated by a compatible version of Tree-sitter.
func (l *Language) Version() int {
	return int(C.ts_language_version(l.c()))
}

// NextState returns the next parse state. Combine this with lookahead iterators to generate
// completion suggestions or valid symbols in error nodes. Use `ts_node_grammar_symbol`
// for valid symbols.
func (l *Language) NextState(curr StateID, sym Symbol) StateID {
	return C.ts_language_next_state(l.c(), curr, sym)
}

func (l *Language) c() *C.TSLanguage {
	return (*C.TSLanguage)(l.ptr)
}

func (e LanguageError) Error() string {
	return fmt.Sprintf("Incompatible language version %d. Expected minimum %d, maximum %d",
		e, C.TREE_SITTER_MIN_COMPATIBLE_LANGUAGE_VERSION, C.TREE_SITTER_LANGUAGE_VERSION)
}
