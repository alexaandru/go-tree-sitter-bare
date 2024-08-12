package sitter

// #include "sitter.h"
import "C"

import (
	"sync"
	"unsafe"
)

// LookaheadIterator holds the pointer to the corresponding C struct.
type LookaheadIterator struct {
	ptr  unsafe.Pointer
	once sync.Once
}

// NewLookaheadIterator creates a new lookahead iterator for the given language and parse state.
//
// This returns `NULL` if state is invalid for the language.
//
// Repeatedly using `ts_lookahead_iterator_next` and
// `ts_lookahead_iterator_current_symbol` will generate valid symbols in the
// given parse state. Newly created lookahead iterators will contain the `ERROR`
// symbol.
//
// Lookahead iterators can be useful to generate suggestions and improve syntax
// error diagnostics. To get symbols valid in an ERROR node, use the lookahead
// iterator on its first leaf node state. For `MISSING` nodes, a lookahead
// iterator created on the previous non-extra leaf node may be appropriate.
func NewLookaheadIterator(lang *Language, stateID StateID) *LookaheadIterator {
	ptr := C.ts_lookahead_iterator_new(lang.c(), stateID)
	if ptr == nil || uintptr(unsafe.Pointer(ptr)) == 0 {
		return nil
	}

	return &LookaheadIterator{ptr: unsafe.Pointer(ptr)}
}

// Delete deletes a lookahead iterator freeing all the memory used.
func (iter *LookaheadIterator) Delete() {
	iter.once.Do(func() { C.ts_lookahead_iterator_delete(iter.c()) })
}

// ResetState resets the lookahead iterator to another state.
//
// This returns `true` if the iterator was reset to the given state and `false`
// otherwise.
func (iter *LookaheadIterator) ResetState(stateID StateID) bool {
	return bool(C.ts_lookahead_iterator_reset_state(iter.c(), stateID))
}

// Reset resets the lookahead iterator.
//
// This returns `true` if the language was set successfully and `false`
// otherwise.
func (iter *LookaheadIterator) Reset(lang *Language, stateID StateID) bool {
	return bool(C.ts_lookahead_iterator_reset(iter.c(), lang.c(), stateID))
}

// Language returns the current language of the lookahead iterator.
func (iter *LookaheadIterator) Language() *Language {
	return NewLanguage(unsafe.Pointer(C.ts_lookahead_iterator_language(iter.c())))
}

// Next advances the lookahead iterator to the next symbol.
//
// This returns `true` if there is a new symbol and `false` otherwise.
func (iter *LookaheadIterator) Next() bool {
	return bool(C.ts_lookahead_iterator_next(iter.c()))
}

// CurrentSymbol returns the current symbol of the lookahead iterator.
func (iter *LookaheadIterator) CurrentSymbol() Symbol {
	return C.ts_lookahead_iterator_current_symbol(iter.c())
}

// CurrentSymbolName returns the current symbol type of the lookahead iterator
// as a null terminated string.
func (iter *LookaheadIterator) CurrentSymbolName() string {
	return C.GoString(C.ts_lookahead_iterator_current_symbol_name(iter.c()))
}

func (iter *LookaheadIterator) c() *C.TSLookaheadIterator {
	return (*C.TSLookaheadIterator)(iter.ptr)
}
