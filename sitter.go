package sitter

import (
	"context"
	"unsafe"
)

// NewLanguage creates new Language from c pointer.
func NewLanguage(ptr unsafe.Pointer) *Language {
	return &Language{ptr}
}

// Parse is a shortcut for parsing bytes of source code, returns root node.
//
// Deprecated: use ParseCtx instead.
func Parse(content []byte, lang *Language) (*Node, error) {
	return ParseCtx(context.Background(), content, lang)
}

// ParseCtx is a shortcut for parsing bytes of source code, returns root node.
func ParseCtx(ctx context.Context, content []byte, lang *Language) (*Node, error) {
	p := NewParser()
	p.SetLanguage(lang)

	tree, err := p.ParseCtx(ctx, nil, content)
	if err != nil {
		return nil, err
	}

	return tree.RootNode(), nil
}
