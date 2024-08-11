//go:generate stringer -type=SymbolType -trimprefix=SymbolType -output=stringer1.go .
//go:generate stringer -type=IterMode -output=stringer2.go .
//go:generate stringer -type=QueryError -trimprefix=QueryError -output=stringer3.go .
//go:generate stringer -type=QueryPredicateStepType -trimprefix=QueryPredicateStepType -output=stringer4.go .

package sitter

// #include "sitter.h"
import "C"

import (
	"context"
	"unsafe"
)

//nolint:revive,stylecheck // ok
const (
	TREE_SITTER_LANGUAGE_VERSION                = int(C.TREE_SITTER_LANGUAGE_VERSION)
	TREE_SITTER_MIN_COMPATIBLE_LANGUAGE_VERSION = int(C.TREE_SITTER_MIN_COMPATIBLE_LANGUAGE_VERSION)
)

// Parse is a shortcut for parsing bytes of source code, returns root node.
func Parse(ctx context.Context, content []byte, lang *Language) (n *Node, err error) {
	p := NewParser()
	p.SetLanguage(lang)

	tree, err := p.ParseString(ctx, nil, content)
	if err != nil {
		return
	}

	return tree.RootNode(), nil
}

// NewLanguage initializes a new language from the provided pointer.
func NewLanguage(ptr unsafe.Pointer) (l *Language) {
	if ptr == nil {
		return
	}

	return &Language{ptr: ptr}
}
