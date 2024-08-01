package sitter

// #include "sitter.h"
import "C"

import (
	"sync"
	"unsafe"
)

// Language defines how to parse a particular programming language
type Language struct {
	ptr  unsafe.Pointer
	once sync.Once
}

// StateID is used for parser state ID.
type StateID = C.TSStateId

// FieldID is used for parser field ID.
type FieldID = C.TSFieldId

// Copy returns another reference to the given language.
func (l *Language) Copy() *Language {
	return newLanguage(C.ts_language_copy((*C.TSLanguage)(l.ptr)))
}

// Delete frees any dynamically-allocated resources for this language, if
// this is the last reference.
func (l *Language) Delete() {
	l.once.Do(func() {
		C.ts_language_delete((*C.TSLanguage)(l.ptr))
		l.ptr = nil
		l = nil
	})
}

// SymbolCount returns the number of distinct field names in the language.
func (l *Language) SymbolCount() uint32 {
	return uint32(C.ts_language_symbol_count((*C.TSLanguage)(l.ptr)))
}

// StateCount returns the number of valid states in the language.
func (l *Language) StateCount() uint32 {
	return uint32(C.ts_language_state_count((*C.TSLanguage)(l.ptr)))
}

// SymbolName returns a node type string for the given Symbol.
func (l *Language) SymbolName(s Symbol) string {
	return C.GoString(C.ts_language_symbol_name((*C.TSLanguage)(l.ptr), s))
}

// SymbolID returns the numerical id for the given node type string.
func (l *Language) SymbolID(name string, isNamed bool) Symbol {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	return C.ts_language_symbol_for_name((*C.TSLanguage)(l.ptr), cName, C.uint(len(name)), C._Bool(isNamed))
}

// FieldCount returns the number of distinct field names in the language.
func (l *Language) FieldCount() uint32 {
	return uint32(C.ts_language_field_count((*C.TSLanguage)(l.ptr)))
}

// FieldName returns the field name string for the given numerical id.
func (l *Language) FieldName(idx int) string {
	return C.GoString(C.ts_language_field_name_for_id((*C.TSLanguage)(l.ptr), C.ushort(idx)))
}

// FieldID returns the numerical id for the given field name string.
func (l *Language) FieldID(name string) FieldID {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	return C.ts_language_field_id_for_name((*C.TSLanguage)(l.ptr), cName, C.uint(len(name)))
}

// SymbolType returns named, anonymous, or a hidden type for a Symbol.
func (l *Language) SymbolType(s Symbol) Symbol {
	return Symbol(C.ts_language_symbol_type((*C.TSLanguage)(l.ptr), s))
}

// Version returns the ABI version number for this language. This version number is used
// to ensure that languages were generated by a compatible version of Tree-sitter.
func (l *Language) Version() uint32 {
	return uint32(C.ts_language_version((*C.TSLanguage)(l.ptr)))
}

// NextState returns the next parse state. Combine this with lookahead iterators to generate
// completion suggestions or valid symbols in error nodes. Use `ts_node_grammar_symbol`
// for valid symbols.
func (l *Language) NextState(curr StateID, sym Symbol) StateID {
	return C.ts_language_next_state((*C.TSLanguage)(l.ptr), curr, sym)
}
