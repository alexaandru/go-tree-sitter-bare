package sitter

// #include "sitter.h"
import "C"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Parser produces concrete syntax tree based on source code using Language.
type Parser struct {
	c      *C.TSParser
	cancel *uint64
}

// Input type lets you specify how to read the text.
type Input struct {
	// Payload an arbitrary pointer that will be passed to each invocation
	// of the Read function. NOTE: I do see it passed into ts_parser_parse()
	// however I don't see it used. Just keeping it here for documenting purposes.
	Payload unsafe.Pointer
	//  Read is a function to retrieve a chunk of text at a given byte offset
	//  and (row, column) position. The function should return a pointer to the
	//  text and write its length to the [`bytes_read`] pointer. The parser does
	//  not take ownership of this buffer; it just borrows it until it has
	//  finished reading it. The function should write a zero value to the
	//  [`bytes_read`] pointer to indicate the end of the document.
	Read ReadFunc
	// Encoding is an indication of how the text is encoded.
	// Either `TSInputEncodingUTF8` or `TSInputEncodingUTF16`.
	Encoding InputEncoding
}

// InputEncoding indicates the encoding of the text to parse.
type InputEncoding = C.TSInputEncoding

// ReadFunc is a function to retrieve a chunk of text at a given byte offset and
// (row, column) position. It should return nil to indicate the end of the document.
type ReadFunc func(offset uint32, position Point) []byte

// keeps callbacks for parser.parse method
type readFuncsMap struct {
	funcs map[int]ReadFunc
	index int
	sync.RWMutex
}

// Input encoding types.
const (
	InputEncodingUTF8  = C.TSInputEncodingUTF8
	InputEncodingUTF16 = C.TSInputEncodingUTF16
)

// Maintain a map of read functions that can be called from C.
var readFuncs = &readFuncsMap{funcs: map[int]ReadFunc{}} //nolint:gochecknoglobals // ok

// Possible error types.
var (
	ErrOperationLimit = errors.New("operation limit was hit")
	ErrNoLanguage     = errors.New("cannot parse without language")
)

// NewParser creates a new Parser.
func NewParser() (p *Parser) {
	cancel := uint64(0)
	p = &Parser{c: C.ts_parser_new()}

	p.SetCancellationFlag(&cancel)

	runtime.AddCleanup(p, func(c *C.TSParser) { C.ts_parser_delete(c) }, p.c)

	return p
}

// Language returns the parser's current language, if set.
func (p *Parser) Language() (_ *Language) {
	return NewLanguage(unsafe.Pointer(C.ts_parser_language(p.c)))
}

// SetLanguage sets the language that the parser should use for parsing.
//
// Returns a boolean indicating whether or not the language was successfully
// assigned. True means assignment succeeded. False means there was a version
// mismatch: the language was generated with an incompatible version of the
// Tree-sitter CLI. Check the language's version using `ts_language_version`
// and compare it to this library's `TREE_SITTER_LANGUAGE_VERSION` and
// `TREE_SITTER_MIN_COMPATIBLE_LANGUAGE_VERSION` constants.
func (p *Parser) SetLanguage(lang *Language) bool {
	cLang := (*C.struct_TSLanguage)(lang.ptr)
	return bool(C.ts_parser_set_language(p.c, cLang))
}

// SetIncludedRanges sets the ranges of text that the parser should include when parsing.
//
// By default, the parser will always include entire documents. This function
// allows you to parse only a *portion* of a document but still return a syntax
// tree whose ranges match up with the document as a whole. You can also pass
// multiple disjoint ranges.
//
// The second and third parameters specify the location and length of an array
// of ranges. The parser does *not* take ownership of these ranges; it copies
// the data, so it doesn't matter how these ranges are allocated.
//
// If `count` is zero, then the entire document will be parsed. Otherwise,
// the given ranges must be ordered from earliest to latest in the document,
// and they must not overlap. That is, the following must hold for all:
//
// `i < count - 1`: `ranges[i].end_byte <= ranges[i + 1].start_byte`
//
// If this requirement is not satisfied, the operation will fail, the ranges
// will not be assigned, and this function will return `false`. On success,
// this function returns `true`.
func (p *Parser) SetIncludedRanges(ranges []Range) bool {
	cRanges := make([]C.TSRange, len(ranges))
	for i, r := range ranges {
		cRanges[i] = r.c()
	}

	return bool(C.ts_parser_set_included_ranges(p.c, (*C.TSRange)(unsafe.Pointer(&cRanges[0])), C.uint(len(ranges))))
}

// IncludedRanges returns the ranges of text that the parser will include when parsing.
//
// The returned pointer is owned by the parser. The caller should not free it
// or write to it.
func (p *Parser) IncludedRanges() (out []Range) {
	count := C.uint(0)
	pp := C.ts_parser_included_ranges(p.c, &count)

	return mkRanges(pp, count)
}

// Parse produces new Tree by reading from a callback defined in input
// it is useful if your data is stored in specialized data structure
// as it will avoid copying the data into []bytes and faster access
// to edited part of the data.
//
// If you are parsing this document for the first time, pass `NULL` for the
// `old_tree` parameter. Otherwise, if you have already parsed an earlier
// version of this document and the document has since been edited, pass the
// previous syntax tree so that the unchanged parts of it can be reused.
// This will save time and memory. For this to work correctly, you must have
// already edited the old syntax tree using the [`ts_tree_edit`] function in a
// way that exactly matches the source code changes.
//
// See Input for details about it.
//
// This function returns a syntax tree on success, and `NULL` on failure. There
// are three possible reasons for failure:
//  1. The parser does not have a language assigned. Check for this using the
//     [`ts_parser_language`] function.
//  2. Parsing was cancelled due to a timeout that was set by an earlier call to
//     the [`ts_parser_set_timeout_micros`] function. You can resume parsing from
//     where the parser left out by calling [`ts_parser_parse`] again with the
//     same arguments. Or you can start parsing from scratch by first calling
//     [`ts_parser_reset`].
//  3. Parsing was cancelled using a cancellation flag that was set by an
//     earlier call to [`ts_parser_set_cancellation_flag`]. You can resume parsing
//     from where the parser left out by calling [`ts_parser_parse`] again with
//     the same arguments.
func (p *Parser) Parse(ctx context.Context, oldTree *Tree, input Input) (*Tree, error) {
	var baseTree *C.TSTree

	if oldTree != nil {
		baseTree = oldTree.c
	}

	funcID := readFuncs.register(input.Read)
	baseTree = C.call_ts_parser_parse(p.c, baseTree, C.int(funcID), input.Encoding)

	readFuncs.unregister(funcID)

	return p.convertTSTree(ctx, baseTree)
}

// ParseString produces new Tree from content (optionally using old tree).
//
// Uses the parser to parse some source code stored in one contiguous buffer.
// If the optional encoding is passed, it will be used for parsing.
// See `Parse()` for further details, as the behavior virtually the same.
func (p *Parser) ParseString(ctx context.Context, oldTree *Tree, content []byte, opts ...InputEncoding) (*Tree, error) {
	var baseTree *C.TSTree

	if oldTree != nil {
		baseTree = oldTree.c
	}

	parseComplete := make(chan struct{})

	// run goroutine only if context is cancelable to avoid performance impact
	if ctx.Done() != nil {
		go func() {
			select {
			case <-ctx.Done():
				atomic.StoreUint64(p.cancel, 1)
			case <-parseComplete:
				return
			}
		}()
	}

	input := C.CBytes(content)

	if len(opts) > 0 {
		baseTree = C.ts_parser_parse_string_encoding(p.c, baseTree, (*C.char)(input), C.uint(len(content)), opts[0])
	} else {
		baseTree = C.ts_parser_parse_string(p.c, baseTree, (*C.char)(input), C.uint(len(content)))
	}

	close(parseComplete)

	C.free(input)

	return p.convertTSTree(ctx, baseTree)
}

// Reset instructs the parser to start the next parse from the beginning.
//
// If the parser previously failed because of a timeout or a cancellation, then
// by default, it will resume where it left off on the next call to
// `ts_parser_parse` or other parsing functions. If you don't want to resume,
// and instead intend to use this parser to parse some other document, you must
// call `ts_parser_reset` first.
func (p *Parser) Reset() {
	C.ts_parser_reset(p.c)
}

// SetTimeoutMicros limits the maximum duration in microseconds that parsing should
// be allowed to take before halting.
//
// If parsing takes longer than this, it will halt early, returning NULL.
// See `ts_parser_parse` for more information.
func (p *Parser) SetTimeoutMicros(limit int) {
	C.ts_parser_set_timeout_micros(p.c, C.uint64_t(limit))
}

// TimeoutMicros returns the duration in microseconds that parsing is allowed to take.
func (p *Parser) TimeoutMicros() int {
	return int(C.ts_parser_timeout_micros(p.c))
}

// SetCancellationFlag sets the parser's current cancellation flag pointer.
//
// If a non-null pointer is assigned, then the parser will periodically read
// from this pointer during parsing. If it reads a non-zero value, it will
// halt early, returning NULL. See [`ts_parser_parse`] for more information.
func (p *Parser) SetCancellationFlag(flag *uint64) {
	p.cancel = flag

	C.ts_parser_set_cancellation_flag(p.c, (*C.size_t)(unsafe.Pointer(p.cancel)))
}

// CancellationFlag returns the parser's current cancellation flag pointer.
func (p *Parser) CancellationFlag() *uint64 {
	// return (*uint64)(unsafe.Pointer(C.ts_parser_cancellation_flag(p.c)))
	return p.cancel
}

// Debug enables debug output to stderr.
func (p *Parser) Debug() {
	p.SetLogger(C.stderr_logger_new(true))
}

// SetLogger sets the logger that a parser should use during parsing.
//
// The parser does not take ownership over the logger payload. If a logger was
// previously assigned, the caller is responsible for releasing any memory
// owned by the previous logger.
func (p *Parser) SetLogger(logger C.TSLogger) {
	C.ts_parser_set_logger(p.c, logger)
}

// Logger returns the parser's current logger.
func (p *Parser) Logger() C.TSLogger {
	return C.ts_parser_logger(p.c)
}

// PrintDotGraphs can be used to write debugging graphs during parsing.
//
// The graphs are formatted in the DOT language. You may want
// to pipe these graphs directly to a `dot(1)` process in order to generate
// SVG output. You can turn off this logging by passing a negative number.
func (p *Parser) PrintDotGraphs(name string) (err error) {
	f, err := os.Create(name)
	if err != nil {
		return
	}

	C.ts_parser_print_dot_graphs(p.c, C.int(f.Fd()))

	if err = f.Close(); err != nil {
		err = fmt.Errorf("cannot save dot file: %w", err)
	}

	return
}

// converts the tree-sitter response into a *Tree or an error.
//
// tree-sitter can fail for 3 reasons:
// - cancelation
// - operation limit hit
// - no language set
//
// We check for all those conditions if there return value is nil.
// See `Parse()` comment for further details.
func (p *Parser) convertTSTree(ctx context.Context, tsTree *C.TSTree) (t *Tree, err error) {
	if tsTree == nil {
		if err = ctx.Err(); err != nil {
			// reset cancellation flag so the parse can be re-used
			atomic.StoreUint64(p.cancel, 0)

			// context cancellation caused a timeout, return that error
			return nil, fmt.Errorf("failed converting TSTree -> Tree: %w", err)
		}

		if p.Language() == nil {
			return nil, ErrNoLanguage
		}

		return nil, ErrOperationLimit
	}

	return newTree(tsTree), nil
}

func (m *readFuncsMap) register(f ReadFunc) (id int) {
	m.Lock()
	defer m.Unlock()

	m.index++

	id = m.index
	m.funcs[id] = f

	return
}

func (m *readFuncsMap) unregister(id int) {
	m.Lock()
	defer m.Unlock()

	delete(m.funcs, id)
}

func (m *readFuncsMap) get(id int) ReadFunc {
	m.RLock()
	defer m.RUnlock()

	return m.funcs[id]
}

//export callReadFunc
func callReadFunc(id C.int, byteIndex C.uint32_t, pos C.TSPoint, bytesRead *C.uint32_t) *C.char {
	readFunc := readFuncs.get(int(id))
	content := readFunc(uint32(byteIndex), mkPoint(pos))
	*bytesRead = C.uint32_t(len(content))

	// Note: This memory is freed inside the C code; see sitter.c
	input := C.CBytes(content)

	return (*C.char)(input)
}
