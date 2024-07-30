package sitter

// #include "bindings.h"
import "C" //nolint:gocritic // ok

import "unsafe" //nolint:gocritic // ok

// Language defines how to parse a particular programming language
type Language struct {
	ptr unsafe.Pointer
}

// SymbolName returns a node type string for the given Symbol.
func (l *Language) SymbolName(s Symbol) string {
	return C.GoString(C.ts_language_symbol_name((*C.TSLanguage)(l.ptr), s))
}

// SymbolType returns named, anonymous, or a hidden type for a Symbol.
func (l *Language) SymbolType(s Symbol) SymbolType {
	return SymbolType(C.ts_language_symbol_type((*C.TSLanguage)(l.ptr), s))
}

// SymbolCount returns the number of distinct field names in the language.
func (l *Language) SymbolCount() uint32 {
	return uint32(C.ts_language_symbol_count((*C.TSLanguage)(l.ptr)))
}

// FieldName returns the field name string for the given numerical id.
func (l *Language) FieldName(idx int) string {
	return C.GoString(C.ts_language_field_name_for_id((*C.TSLanguage)(l.ptr), C.ushort(idx)))
}
