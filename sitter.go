package sitter

// #include "sitter.h"
import "C" //nolint:gocritic // ok

import (
	"context"
	"unsafe" //nolint:gocritic // ok
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

// mkRange constructs a Range from a C TSRange.
func mkRange(r C.TSRange) Range {
	return Range{
		StartPoint: Point{Row: uint32(r.start_point.row), Column: uint32(r.start_point.column)},
		EndPoint:   Point{Row: uint32(r.end_point.row), Column: uint32(r.end_point.column)},
		StartByte:  uint32(r.start_byte),
		EndByte:    uint32(r.end_byte),
	}
}

// mkCRange constructs a C TSRange from a Range.
func mkCRange(r Range) C.TSRange {
	return C.TSRange{
		start_point: C.TSPoint{row: C.uint32_t(r.StartPoint.Row), column: C.uint32_t(r.StartPoint.Column)},
		end_point:   C.TSPoint{row: C.uint32_t(r.EndPoint.Row), column: C.uint32_t(r.EndPoint.Column)},
		start_byte:  C.uint32_t(r.StartByte),
		end_byte:    C.uint32_t(r.EndByte),
	}
}

// FIXME: When (and how) to free memory for the C array?
// The returned Go slice is backed by the C array. So the
// C array should be freed when the Go slice is GC'ed:
// need to use runtime.SetFinalizer() "somehow" (TBD).
func mkRanges(p *C.TSRange, count C.uint32_t) (out []Range) {
	out = make([]Range, count)

	for i, r := range unsafe.Slice(p, int(count)) {
		out[i] = mkRange(r)
	}

	return
}
