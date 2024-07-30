package sitter //nolint:gocritic // ok

// #include "sitter.h"
import "C" //nolint:gocritic // ok

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe" //nolint:gocritic // ok
)

// Parser produces concrete syntax tree based on source code using Language
type Parser struct {
	c        *C.TSParser
	cancel   *uintptr
	isClosed bool
}

// Input defines parameters for parse method
type Input struct {
	Read     ReadFunc
	Encoding InputEncoding
}

// InputEncoding is a encoding of the text to parse
type InputEncoding int

// ReadFunc is a function to retrieve a chunk of text at a given byte offset and (row, column) position
// it should return nil to indicate the end of the document
type ReadFunc func(offset uint32, position Point) []byte

// keeps callbacks for parser.parse method
type readFuncsMap struct {
	funcs map[int]ReadFunc
	count int

	sync.Mutex
}

// Input encoding types.
const (
	InputEncodingUTF8 InputEncoding = iota
	InputEncodingUTF16
)

// maintain a map of read functions that can be called from C
var readFuncs = &readFuncsMap{funcs: map[int]ReadFunc{}}

// Possible error types.
var (
	ErrOperationLimit = errors.New("operation limit was hit")
	ErrNoLanguage     = errors.New("cannot parse without language")
)

// NewParser creates new Parser
func NewParser() *Parser {
	cancel := uintptr(0)
	p := &Parser{c: C.ts_parser_new(), cancel: &cancel}

	C.ts_parser_set_cancellation_flag(p.c, (*C.size_t)(unsafe.Pointer(p.cancel)))

	runtime.SetFinalizer(p, (*Parser).Close)

	return p
}

// SetLanguage assignes Language to a parser
func (p *Parser) SetLanguage(lang *Language) {
	cLang := (*C.struct_TSLanguage)(lang.ptr)

	C.ts_parser_set_language(p.c, cLang)
}

// Parse produces new Tree from content using old tree
//
// Deprecated: use ParseCtx instead.
func (p *Parser) Parse(oldTree *Tree, content []byte) (*Tree, error) {
	return p.ParseCtx(context.Background(), oldTree, content)
}

// ParseCtx produces new Tree from content using old tree
func (p *Parser) ParseCtx(ctx context.Context, oldTree *Tree, content []byte) (*Tree, error) {
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
				atomic.StoreUintptr(p.cancel, 1)
			case <-parseComplete:
				return
			}
		}()
	}

	input := C.CBytes(content)
	baseTree = C.ts_parser_parse_string(p.c, baseTree, (*C.char)(input), C.uint32_t(len(content)))

	close(parseComplete)

	C.free(input)

	return p.convertTSTree(ctx, baseTree)
}

// ParseInput produces new Tree by reading from a callback defined in input
// it is useful if your data is stored in specialized data structure
// as it will avoid copying the data into []bytes
// and faster access to edited part of the data
//
// Deprecated: use ParseInputCtx instead.
func (p *Parser) ParseInput(oldTree *Tree, input Input) (*Tree, error) {
	return p.ParseInputCtx(context.Background(), oldTree, input)
}

// ParseInputCtx produces new Tree by reading from a callback defined in input
// it is useful if your data is stored in specialized data structure
// as it will avoid copying the data into []bytes
// and faster access to edited part of the data
func (p *Parser) ParseInputCtx(ctx context.Context, oldTree *Tree, input Input) (*Tree, error) {
	var baseTree *C.TSTree

	if oldTree != nil {
		baseTree = oldTree.c
	}

	funcID := readFuncs.register(input.Read)
	baseTree = C.call_ts_parser_parse(p.c, baseTree, C.int(funcID), C.TSInputEncoding(input.Encoding))

	readFuncs.unregister(funcID)

	return p.convertTSTree(ctx, baseTree)
}

// OperationLimit returns the duration in microseconds that parsing is allowed to take
func (p *Parser) OperationLimit() int {
	return int(C.ts_parser_timeout_micros(p.c))
}

// SetOperationLimit limits the maximum duration in microseconds that parsing should be allowed to take before halting
func (p *Parser) SetOperationLimit(limit int) {
	C.ts_parser_set_timeout_micros(p.c, C.uint64_t(limit))
}

// Reset causes the parser to parse from scratch on the next call to parse, instead of resuming
// so that it sees the changes to the beginning of the source code.
func (p *Parser) Reset() {
	C.ts_parser_reset(p.c)
}

// SetIncludedRanges sets text ranges of a file
func (p *Parser) SetIncludedRanges(ranges []Range) {
	cRanges := make([]C.TSRange, len(ranges))

	for i, r := range ranges {
		cRanges[i] = C.TSRange{
			start_point: C.TSPoint{
				row:    C.uint32_t(r.StartPoint.Row),
				column: C.uint32_t(r.StartPoint.Column),
			},
			end_point: C.TSPoint{
				row:    C.uint32_t(r.EndPoint.Row),
				column: C.uint32_t(r.EndPoint.Column),
			},
			start_byte: C.uint32_t(r.StartByte),
			end_byte:   C.uint32_t(r.EndByte),
		}
	}

	C.ts_parser_set_included_ranges(p.c, (*C.TSRange)(unsafe.Pointer(&cRanges[0])), C.uint(len(ranges)))
}

// Debug enables debug output to stderr
func (p *Parser) Debug() {
	logger := C.stderr_logger_new(true)

	C.ts_parser_set_logger(p.c, logger)
}

// Close should be called to ensure that all the memory used by the parse is freed.
//
// As the constructor in go-tree-sitter would set this func call through runtime.SetFinalizer,
// parser.Close() will be called by Go's garbage collector and users would not have to call this manually.
func (p *Parser) Close() {
	if !p.isClosed {
		C.ts_parser_delete(p.c)
	}

	p.isClosed = true
}

// convertTSTree converts the tree-sitter response into a *Tree or an error.
//
// tree-sitter can fail for 3 reasons:
// - cancelation
// - operation limit hit
// - no language set
//
// We check for all those conditions if there return value is nil.
// see: https://github.com/tree-sitter/tree-sitter/blob/7890a29db0b186b7b21a0a95d99fa6c562b8316b/lib/include/tree_sitter/api.h#L209-L246
func (p *Parser) convertTSTree(ctx context.Context, tsTree *C.TSTree) (*Tree, error) {
	if tsTree == nil {
		if ctx.Err() != nil {
			// reset cancellation flag so the parse can be re-used
			atomic.StoreUintptr(p.cancel, 0)

			// context cancellation caused a timeout, return that error
			return nil, ctx.Err()
		}

		if C.ts_parser_language(p.c) == nil {
			return nil, ErrNoLanguage
		}

		return nil, ErrOperationLimit
	}

	return p.newTree(tsTree), nil
}

func (m *readFuncsMap) register(f ReadFunc) int {
	m.Lock()
	defer m.Unlock()

	m.count++
	m.funcs[m.count] = f

	return m.count
}

func (m *readFuncsMap) unregister(id int) {
	m.Lock()
	defer m.Unlock()

	delete(m.funcs, id)
}

func (m *readFuncsMap) get(id int) ReadFunc {
	m.Lock()
	defer m.Unlock()

	return m.funcs[id]
}

//export callReadFunc
func callReadFunc(id C.int, byteIndex C.uint32_t, position C.TSPoint, bytesRead *C.uint32_t) *C.char {
	readFunc := readFuncs.get(int(id))
	content := readFunc(uint32(byteIndex), Point{
		Row:    uint32(position.row),
		Column: uint32(position.column),
	})
	*bytesRead = C.uint32_t(len(content))

	// Note: This memory is freed inside the C code; see bindings.c
	input := C.CBytes(content)

	return (*C.char)(input)
}
