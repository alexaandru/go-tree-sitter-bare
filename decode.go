package sitter

// #include "api.h"
import "C"

// DecodeFunction reads one code point from a string, returning the number of bytes consumed.
// It writes the code point to the provided pointer, or -1 if the input is invalid.
type DecodeFunction func(data []byte, length uint32, codePoint *int32) uint32

// Function signature for tree-sitter DecodeFunction
type tsDecodeFunction = C.DecodeFunction

// DecodeUTF8 is the default UTF-8 decoder for use with InputEncodingCustom
//
//nolint:mnd // ok
func DecodeUTF8(data []byte, length uint32, codePoint *int32) (size uint32) {
	// Basic UTF-8 decoder implementation
	if length == 0 {
		*codePoint = -1
		return 0
	}

	b := data[0] //nolint:varnamelen // ok

	// Single byte (ASCII)
	if b&0x80 == 0 {
		*codePoint = int32(b)
		return 1
	}

	switch {
	case b&0xE0 == 0xC0:
		size = 2
	case b&0xF0 == 0xE0:
		size = 3
	case b&0xF8 == 0xF0:
		size = 4
	default:
		// Invalid UTF-8
		*codePoint = -1
		return 1
	}

	if length < size {
		*codePoint = -1
		return 1
	}

	var cp int32

	switch size {
	case 2:
		cp = int32(b&0x1F)<<6 | int32(data[1]&0x3F)
	case 3:
		cp = int32(b&0x0F)<<12 | int32(data[1]&0x3F)<<6 | int32(data[2]&0x3F)
	case 4:
		cp = int32(b&0x07)<<18 | int32(data[1]&0x3F)<<12 | int32(data[2]&0x3F)<<6 | int32(data[3]&0x3F)
	}

	*codePoint = cp

	return size
}
