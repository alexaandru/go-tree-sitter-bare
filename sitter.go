package sitter

// #include "sitter.h"
import "C" //nolint:gocritic // ok

import (
	"context"
	"unsafe" //nolint:gocritic // ok
)

//nolint:revive,stylecheck // ok
const (
	TREE_SITTER_LANGUAGE_VERSION                = C.TREE_SITTER_LANGUAGE_VERSION
	TREE_SITTER_MIN_COMPATIBLE_LANGUAGE_VERSION = C.TREE_SITTER_MIN_COMPATIBLE_LANGUAGE_VERSION
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

func newLanguage[T any, P *T](ptr P) (l *Language) {
	if ptr == nil {
		return
	}

	return &Language{ptr: unsafe.Pointer(ptr)}
}
