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

// mkRange constructs a Range from a C TSRange.
func mkRange(r C.TSRange) Range {
	return Range{
		StartPoint: mkPoint(r.start_point),
		EndPoint:   mkPoint(r.end_point),
		StartByte:  uint32(r.start_byte),
		EndByte:    uint32(r.end_byte),
	}
}

// mkCRange constructs a C TSRange from a Range.
func mkCRange(r Range) C.TSRange {
	return C.TSRange{
		start_point: mkCPoint(r.StartPoint),
		end_point:   mkCPoint(r.EndPoint),
		start_byte:  C.uint32_t(r.StartByte),
		end_byte:    C.uint32_t(r.EndByte),
	}
}

func mkRanges(p *C.TSRange, count C.uint32_t) (out []Range) {
	out = make([]Range, count)

	for i, r := range unsafe.Slice(p, int(count)) {
		out[i] = mkRange(r)
	}

	return
}

func mkPoint(p C.TSPoint) Point {
	return Point{Row: uint32(p.row), Column: uint32(p.column)}
}

func mkCPoint(p Point) C.TSPoint {
	return C.TSPoint{row: C.uint32_t(p.Row), column: C.uint32_t(p.Column)}
}

func newLanguage[T any, P *T](ptr P) (l *Language) {
	if ptr == nil {
		return
	}

	return &Language{ptr: unsafe.Pointer(ptr)}
}
