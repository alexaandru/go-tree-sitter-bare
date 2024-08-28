package sitter

// #include "sitter.h"
import "C"

import (
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

// Query is a tree query, compiled from a string of S-expressions. The query
// itself is immutable. The mutable state used in the process of executing the
// query is stored in a `TSQueryCursor`.
type Query struct {
	c    *C.TSQuery
	once sync.Once
}

// QueryCursor is a stateful struct used to execute a query on a tree.
type QueryCursor struct {
	c    *C.TSQueryCursor
	t    *Tree
	q    *Query // NOTE: Keep a pointer to it to avoid GC. Maybe use Pinner instead?
	once sync.Once
}

// QueryCapture is a captured node by a query with an index.
type QueryCapture struct {
	Node  *Node
	Index uint32
}

// QueryMatch allows you to iterate over the matches.
type QueryMatch struct {
	Captures     []QueryCapture
	ID           uint32
	PatternIndex uint16
}

// QueryPredicateStep represents one step in a predicate.
type QueryPredicateStep struct {
	Type    QueryPredicateStepType
	ValueID uint32
}

// QueryPredicateStepType represents type of step in a predicate.
type QueryPredicateStepType = C.TSQueryPredicateStepType

// QueryPredicateSteps holds all the steps for a predicate.
type QueryPredicateSteps []QueryPredicateStep

// QueryError indicates the type of QueryError.
type QueryError = C.TSQueryError

// DetailedQueryError - if there is an error in the query,
// then the Offset argument will be set to the byte offset of the error,
// and the Type argument will be set to a value that indicates the type of error.
type DetailedQueryError struct {
	Message string
	Type    QueryError
	Offset  uint32
}

// Possible query predicate steps.
const (
	QueryPredicateStepTypeDone    QueryPredicateStepType = C.TSQueryPredicateStepTypeDone
	QueryPredicateStepTypeCapture QueryPredicateStepType = C.TSQueryPredicateStepTypeCapture
	QueryPredicateStepTypeString  QueryPredicateStepType = C.TSQueryPredicateStepTypeString
)

// Possible quantifiers.
const (
	QuantifierZero       = C.TSQuantifierZero
	QuantifierZeroOrOne  = C.TSQuantifierZeroOrOne
	QuantifierZeroOrMore = C.TSQuantifierZeroOrMore
	QuantifierOne        = C.TSQuantifierOne
	QuantifierOneOrMore  = C.TSQuantifierOneOrMore
)

// Error types.
const (
	QueryErrorNone      QueryError = C.TSQueryErrorNone
	QueryErrorSyntax    QueryError = C.TSQueryErrorSyntax
	QueryErrorNodeType  QueryError = C.TSQueryErrorNodeType
	QueryErrorField     QueryError = C.TSQueryErrorField
	QueryErrorCapture   QueryError = C.TSQueryErrorCapture
	QueryErrorStructure QueryError = C.TSQueryErrorStructure
	QueryErrorLanguage  QueryError = C.TSQueryErrorLanguage
)

const (
	maxUint16 = uint16(C.UINT16_MAX)
	maxUint32 = uint32(C.UINT32_MAX)
	// UnlimitedMaxDepth is used for turning off max depth limit for query cursor.
	UnlimitedMaxDepth = maxUint32
)

// Query related errors.
var (
	ErrPredicateArgsWrongCount = errors.New("wrong number of arguments")
	ErrPredicateWrongStart     = errors.New("predicate must begin with a literal value")
	ErrPredicateWrongType      = errors.New("predicate must be a")
)

// NewQuery creates a new query from a string containing one or more S-expression
// patterns. The query is associated with a particular language, and can
// only be run on syntax nodes parsed with that language.
//
// If all of the given patterns are valid, this returns a `TSQuery`.
// If a pattern is invalid, it returns an error which provides two pieces
// of information about the problem:
//  1. The byte offset of the error is written to the `error_offset` parameter.
//  2. The type of error is written to the `error_type` parameter.
func NewQuery(pattern []byte, lang *Language) (q *Query, err error) {
	var (
		erroff  C.uint
		errtype C.TSQueryError
	)

	input := C.CBytes(pattern)
	c := C.ts_query_new( //nolint:varnamelen // ok
		(*C.struct_TSLanguage)(lang.ptr),
		(*C.char)(input),
		C.uint(len(pattern)),
		&erroff,
		&errtype, //nolint:nlreturn // false positive
	)

	C.free(input)

	if errtype != QueryErrorNone {
		return nil, newDetailedQueryError(pattern, errtype, erroff)
	}

	q = &Query{c: c}
	id2str := q.StringValueForID

	for i := range q.PatternCount() {
		for _, steps := range q.PredicatesForPattern(i) {
			if err = steps.assertValid(id2str); err != nil { //nolint:gocritic // ok
				return //nolint:nakedret // ok
			}
		}
	}

	runtime.SetFinalizer(q, (*Query).close)

	return //nolint:nakedret // ok
}

func newDetailedQueryError(pattern []byte, errtype C.TSQueryError, erroff C.uint) *DetailedQueryError {
	errorOffset := uint32(erroff)
	// search for the line containing the offset
	line := 1
	lineStart := 0

	for i, c := range pattern {
		lineStart = i
		if uint32(i) >= errorOffset {
			break
		}
		if c == '\n' {
			line++
		}
	}

	column := int(errorOffset) - lineStart
	errorType := QueryError(errtype) //nolint:unconvert // needed for extra methods

	var message string

	switch errorType {
	// Errors that apply to a single identifier.
	case QueryErrorNodeType, QueryErrorField, QueryErrorCapture:
		// find identifier at input[errorOffset]
		// and report it in the error message
		s := string(pattern[errorOffset:])
		identifierRegexp := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*`)
		m := identifierRegexp.FindStringSubmatch(s)
		if len(m) > 0 {
			message = fmt.Sprintf("invalid %s '%s' at line %d column %d",
				errorType, m[0], line, column)
		} else {
			message = fmt.Sprintf("invalid %s at line %d column %d",
				errorType, line, column)
		}

	// Errors the report position: QueryErrorSyntax, QueryErrorStructure, QueryErrorLanguage.
	default:
		s := string(pattern[errorOffset:])
		lines := strings.Split(s, "\n")
		whitespace := strings.Repeat(" ", column)
		message = fmt.Sprintf("invalid %s at line %d column %d\n%s\n%s^",
			errorType, line, column, lines[0], whitespace)
	}

	return &DetailedQueryError{Offset: errorOffset, Type: errorType, Message: message}
}

// close should be called to ensure that all the memory used by the query is freed.
//
// As the constructor in go-tree-sitter would set this func call through runtime.SetFinalizer,
// parser.close() will be called by Go's garbage collector and users need not call this manually.
func (q *Query) close() {
	q.once.Do(func() { C.ts_query_delete(q.c) })
}

// PatternCount returns the number of patterns in the query.
func (q *Query) PatternCount() uint32 {
	return uint32(C.ts_query_pattern_count(q.c))
}

// CaptureCount returns the number of captures in the query.
func (q *Query) CaptureCount() uint32 {
	return uint32(C.ts_query_capture_count(q.c))
}

// StringCount returns the number of string literals in the query.
func (q *Query) StringCount() uint32 {
	return uint32(C.ts_query_string_count(q.c))
}

// StartByteForPattern returns the byte offset where the given pattern starts
// in the query's source.
//
// This can be useful when combining queries by concatenating their source
// code strings.
func (q *Query) StartByteForPattern(patIdx uint32) uint32 {
	return uint32(C.ts_query_start_byte_for_pattern(q.c, C.uint(patIdx)))
}

// EndByteForPattern returns the byte offset where the given pattern ends
// in the query's source.
//
// This can be useful when combining queries by concatenating their source
// code strings.
func (q *Query) EndByteForPattern(patIdx uint32) uint32 {
	return uint32(C.ts_query_end_byte_for_pattern(q.c, C.uint(patIdx)))
}

// PredicatesForPattern returns all of the predicates for the given pattern in the query.
//
// The predicates are represented as a single array of steps. There are three
// types of steps in this array, which correspond to the three legal values for
// the `type` field:
//   - `TSQueryPredicateStepTypeCapture` - Steps with this type represent names
//     of captures. Their `value_id` can be used with the
//     `ts_query_capture_name_for_id` function to obtain the name of the capture.
//   - `TSQueryPredicateStepTypeString` - Steps with this type represent literal
//     strings. Their `value_id` can be used with the
//     `ts_query_string_value_for_id` function to obtain their string value.
//   - `TSQueryPredicateStepTypeDone` - Steps with this type are *sentinels*
//     that represent the end of an individual predicate. If a pattern has two
//     predicates, then there will be two steps with this `type` in the array.
func (q *Query) PredicatesForPattern(patternIndex uint32) []QueryPredicateSteps {
	var ( //nolint:prealloc // no
		length         C.uint
		predicateSteps QueryPredicateSteps
	)

	cPredicateStep := C.ts_query_predicates_for_pattern(q.c, C.uint(patternIndex), &length)
	cPredicateSteps := unsafe.Slice(cPredicateStep, int(length))

	for _, s := range cPredicateSteps {
		stepType := s._type
		valueID := uint32(s.value_id)
		predicateSteps = append(predicateSteps, QueryPredicateStep{stepType, valueID})
	}

	return predicateSteps.split()
}

// IsPatternRooted checks if the given pattern in the query has a single root node.
func (q *Query) IsPatternRooted(patIdx uint32) bool {
	return bool(C.ts_query_is_pattern_rooted(q.c, C.uint(patIdx)))
}

// IsPatternNonLocal checks if the given pattern in the query is 'non local'.
//
// A non-local pattern has multiple root nodes and can match within a
// repeating sequence of nodes, as specified by the grammar. Non-local
// patterns disable certain optimizations that would otherwise be possible
// when executing a query on a specific range of a syntax tree.
func (q *Query) IsPatternNonLocal(patIdx uint32) bool {
	return bool(C.ts_query_is_pattern_non_local(q.c, C.uint(patIdx)))
}

// IsPatternGuaranteedAtStep checks if a given pattern is guaranteed to
// match once a given step is reached.
// The step is specified by its byte offset in the query's source code.
func (q *Query) IsPatternGuaranteedAtStep(byteOfs uint32) bool {
	return bool(C.ts_query_is_pattern_guaranteed_at_step(q.c, C.uint(byteOfs)))
}

// CaptureNameForID returns the name and length of one of the query's captures,
// or one of the  query's string literals. Each capture and string is associated
// with a  numeric id based on the order that it appeared in the query's source.
func (q *Query) CaptureNameForID(id uint32) string {
	length := C.uint(0)
	name := C.ts_query_capture_name_for_id(q.c, C.uint(id), &length)

	return C.GoStringN(name, C.int(length))
}

// CaptureQuantifierForID returns the quantifier of the query's captures.
// Each capture is associated with a numeric id based on the order that it
// appeared in the query's source.
func (q *Query) CaptureQuantifierForID(id, captureID uint32) C.TSQuantifier {
	return C.ts_query_capture_quantifier_for_id(q.c, C.uint(id), C.uint(captureID))
}

// StringValueForID returns the string value associated with the given query id.
func (q *Query) StringValueForID(id uint32) string {
	length := C.uint(0)
	value := C.ts_query_string_value_for_id(q.c, C.uint(id), &length)

	return C.GoStringN(value, C.int(length))
}

// DisableCapture disables a certain capture within a query.
//
// This prevents the capture from being returned in matches, and also avoids
// any resource usage associated with recording the capture. Currently, there
// is no way to undo this.
func (q *Query) DisableCapture(name string) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	C.ts_query_disable_capture(q.c, cName, C.uint(len(name)))
}

// DisablePattern disables a certain pattern within a query.
//
// This prevents the pattern from matching and removes most of the overhead
// associated with the pattern. Currently, there is no way to undo this.
func (q *Query) DisablePattern(patIdx uint32) {
	C.ts_query_disable_pattern(q.c, C.uint(patIdx))
}

// NewQueryCursor creates a new query cursor.
//
// The cursor stores the state that is needed to iteratively search
// for matches. To use the query cursor, first call `ts_query_cursor_exec`
// to start running a given query on a given syntax node. Then, there are
// two options for consuming the results of the query:
//  1. Repeatedly call `ts_query_cursor_next_match` to iterate over all of the
//     *matches* in the order that they were found. Each match contains the
//     index of the pattern that matched, and an array of captures. Because
//     multiple patterns can match the same set of nodes, one match may contain
//     captures that appear *before* some of the captures from a previous match.
//  2. Repeatedly call `ts_query_cursor_next_capture` to iterate over all of the
//     individual *captures* in the order that they appear. This is useful if
//     don't care about which pattern matched, and just want a single ordered
//     sequence of captures.
//
// If you don't care about consuming all of the results, you can stop calling
// `ts_query_cursor_next_match` or `ts_query_cursor_next_capture` at any point.
//
//	You can then start executing another query on another node by calling
//	`ts_query_cursor_exec` again.
func NewQueryCursor() *QueryCursor {
	qc := &QueryCursor{c: C.ts_query_cursor_new(), t: nil}

	runtime.SetFinalizer(qc, (*QueryCursor).close)

	return qc
}

func newQueryMatch[T, U ~uint16 | ~uint32](id T, patternIndex U) *QueryMatch {
	return &QueryMatch{ID: uint32(id), PatternIndex: uint16(patternIndex)}
}

// close should be called to ensure that all the memory used by the query cursor is freed.
//
// As the constructor in go-tree-sitter would set this func call through runtime.SetFinalizer,
// parser.close() will be called by Go's garbage collector and users need not call this manually.
func (c *QueryCursor) close() {
	c.once.Do(func() { C.ts_query_cursor_delete(c.c) })
}

// Exec executes the query on a given syntax node.
func (c *QueryCursor) Exec(q *Query, n *Node) {
	c.q, c.t = q, n.t
	C.ts_query_cursor_exec(c.c, q.c, n.c)
}

// Manage the maximum number of in-progress matches allowed by this query
// cursor.
//
// Query cursors have an optional maximum capacity for storing lists of
// in-progress captures. If this capacity is exceeded, then the
// earliest-starting match will silently be dropped to make room for further
// matches. This maximum capacity is optional — by default, query cursors allow
// any number of pending matches, dynamically allocating new space for them as
// needed as the query is executed.

// DidExceedMatchLimit see above.
func (c *QueryCursor) DidExceedMatchLimit() bool {
	return bool(C.ts_query_cursor_did_exceed_match_limit(c.c))
}

// MatchLimit see above.
func (c *QueryCursor) MatchLimit() uint32 {
	return uint32(C.ts_query_cursor_match_limit(c.c))
}

// SetMatchLimit see above.
func (c *QueryCursor) SetMatchLimit(limit uint32) {
	C.ts_query_cursor_set_match_limit(c.c, C.uint(limit))
}

// SetByteRange sets the range of bytes in which the query will be executed.
func (c *QueryCursor) SetByteRange(start, end uint32) {
	C.ts_query_cursor_set_byte_range(c.c, C.uint(start), C.uint(end))
}

// SetPointRange sets the range of row/column positions in which the query will be executed.
func (c *QueryCursor) SetPointRange(start, end Point) {
	C.ts_query_cursor_set_point_range(c.c, start.c(), end.c())
}

// NextMatch iterates over matches.
// This function will return (nil, false) when there are no more matches.
// Otherwise, it will populate the QueryMatch with data
// about which pattern matched and which nodes were captured.
func (c *QueryCursor) NextMatch() (*QueryMatch, bool) {
	var cqm C.TSQueryMatch

	if ok := C.ts_query_cursor_next_match(c.c, &cqm); !bool(ok) { //nolint:gocritic // ok
		return nil, false
	}

	qm := newQueryMatch(cqm.id, cqm.pattern_index)

	for _, cc := range unsafe.Slice(cqm.captures, int(cqm.capture_count)) {
		idx := uint32(cc.index)
		node := c.t.cachedNode(cc.node)
		qm.Captures = append(qm.Captures, QueryCapture{Index: idx, Node: node})
	}

	return qm, true
}

// RemoveMatch does smth... TODO
func (c *QueryCursor) RemoveMatch(matchID uint32) {
	C.ts_query_cursor_remove_match(c.c, C.uint(matchID))
}

// NextCapture advances to the next capture of the currently running query.
//
// If there is a capture, write its match to `*match` and its index within
// the matche's capture list to `*capture_index`. Otherwise, return `false`.
func (c *QueryCursor) NextCapture() (qm *QueryMatch, idx uint32, ok bool) {
	var (
		cqm C.TSQueryMatch
		cqi C.uint
	)

	if ok = bool(C.ts_query_cursor_next_capture(c.c, &cqm, &cqi)); !ok { //nolint:gocritic // ok
		return
	}

	qm = newQueryMatch(cqm.id, cqm.pattern_index)

	for _, cc := range unsafe.Slice(cqm.captures, int(cqm.capture_count)) {
		idx2 := uint32(cc.index)
		node := c.t.cachedNode(cc.node)
		qm.Captures = append(qm.Captures, QueryCapture{Index: idx2, Node: node})
	}

	return qm, uint32(cqi), true
}

// SetMaxStartDepth sets the maximum start depth for a query cursor.
//
// This prevents cursors from exploring children nodes at a certain depth.
// Note if a pattern includes many children, then they will still be checked.
//
// The zero max start depth value can be used as a special behavior and
// it helps to destructure a subtree by staying on a node and using captures
// for interested parts. Note that the zero max start depth only limit a search
// depth for a pattern's root node but other nodes that are parts of the pattern
// may be searched at any depth what defined by the pattern structure.
//
// Set to UnlimitedMaxDepth to remove the maximum start depth.
func (c *QueryCursor) SetMaxStartDepth(maxStartDepth uint32) {
	C.ts_query_cursor_set_max_start_depth(c.c, C.uint(maxStartDepth))
}

// Non API.

func (qm *QueryMatch) copy() *QueryMatch {
	return newQueryMatch(qm.ID, qm.PatternIndex)
}

// FilterPredicates filters the given query match with the applicable predicates.
func (c *QueryCursor) FilterPredicates(m *QueryMatch, input []byte) (qm *QueryMatch) { //nolint:funlen,gocognit,cyclop,lll // TODO
	qm = m.copy()

	predicates := c.q.PredicatesForPattern(uint32(qm.PatternIndex))
	if len(predicates) == 0 {
		qm.Captures = m.Captures
		return qm
	}

	// Track if we matched all predicates globally.
	matchedAll := true

	// Check each predicate against the match.
	for _, steps := range predicates {
		if len(steps) == 0 {
			continue
		}

		switch op := c.q.StringValueForID(steps[0].ValueID); op {
		case "eq?", "not-eq?":
			isPositive := op == "eq?"
			expectedCaptureNameLeft := c.q.CaptureNameForID(steps[1].ValueID)

			if steps[2].Type == QueryPredicateStepTypeCapture { //nolint:nestif // ok
				expectedCaptureNameRight := c.q.CaptureNameForID(steps[2].ValueID)

				var nodeLeft, nodeRight *Node

				for _, cpt := range m.Captures {
					captureName := c.q.CaptureNameForID(cpt.Index)

					if captureName == expectedCaptureNameLeft {
						nodeLeft = cpt.Node
					}

					if captureName == expectedCaptureNameRight {
						nodeRight = cpt.Node
					}

					if nodeLeft != nil && nodeRight != nil {
						if (nodeLeft.Content(input) == nodeRight.Content(input)) != isPositive {
							matchedAll = false
						}

						break
					}
				}
			} else {
				expectedValueRight := c.q.StringValueForID(steps[2].ValueID)

				for _, cpt := range m.Captures {
					captureName := c.q.CaptureNameForID(cpt.Index)

					if expectedCaptureNameLeft != captureName {
						continue
					}

					if (cpt.Node.Content(input) == expectedValueRight) != isPositive {
						matchedAll = false
						break
					}
				}
			}
		case "match?", "not-match?":
			isPositive := op == "match?"
			expectedCaptureName := c.q.CaptureNameForID(steps[1].ValueID)
			regex := regexp.MustCompile(c.q.StringValueForID(steps[2].ValueID))

			for _, cpt := range m.Captures {
				captureName := c.q.CaptureNameForID(cpt.Index)
				if expectedCaptureName != captureName {
					continue
				}

				if regex.MatchString(cpt.Node.Content(input)) != isPositive {
					matchedAll = false
					break
				}
			}
		}

		if !matchedAll {
			break
		}
	}

	if matchedAll {
		qm.Captures = append(qm.Captures, m.Captures...)
	}

	return //nolint:nakedret // ok
}

func (err *DetailedQueryError) Error() string {
	return err.Message
}

func (steps QueryPredicateSteps) split() (out []QueryPredicateSteps) {
	currentSteps := make(QueryPredicateSteps, 0, len(steps))

	for _, step := range steps {
		currentSteps = append(currentSteps, step)
		if step.Type == QueryPredicateStepTypeDone {
			out = append(out, currentSteps)
			currentSteps = QueryPredicateSteps{}
		}
	}

	return
}

func (steps QueryPredicateSteps) assertValid(valueFn func(uint32) string) (err error) {
	if len(steps) == 0 {
		return
	}

	if steps[0].Type != QueryPredicateStepTypeString {
		return ErrPredicateWrongStart
	}

	var errx [3]error

	//nolint:mnd // ok
	switch op := valueFn(steps[0].ValueID); op {
	case "eq?", "not-eq?":
		errx[0] = steps.assertCount(op, 4)
		errx[1] = steps.assertStepType(op, 1, QueryPredicateStepTypeCapture, valueFn)
	case "match?", "not-match?":
		errx[0] = steps.assertCount(op, 4)
		errx[1] = steps.assertStepType(op, 1, QueryPredicateStepTypeCapture, valueFn)
		errx[2] = steps.assertStepType(op, 2, QueryPredicateStepTypeString, valueFn)
	case "set!", "is?", "is-not?":
		errx[0] = steps.assertCount(op, 3, 4)
		errx[1] = steps.assertStepType(op, 1, QueryPredicateStepTypeString, valueFn)
		errx[2] = steps.assertStepType(op, 2, QueryPredicateStepTypeString, valueFn)
	}

	return cmp.Or(errx[:]...) //nolint:wrapcheck // ok, the actual errors are wrapped
}

func (steps QueryPredicateSteps) assertCount(op string, expCount int, opts ...int) (err error) {
	switch len(opts) {
	case 0:
		if x := len(steps); x != expCount {
			err = fmt.Errorf("%w to `#%s` predicate. Expected %d, got %d",
				ErrPredicateArgsWrongCount, op, expCount-2, x-2) //nolint:mnd // ok
		}
	default:
		expMax := opts[0]
		if x := len(steps); x < expCount || x > expMax {
			err = fmt.Errorf("%w to `#%s` predicate. Expected %d or %d, got %d",
				ErrPredicateArgsWrongCount, op, expCount-2, expMax-2, x-2) //nolint:mnd // ok
		}
	}

	return
}

func (steps QueryPredicateSteps) assertStepType(op string, step int, expType QueryPredicateStepType, valueFn func(uint32) string) (err error) { //nolint:lll // ok
	// Only validate step if exists, to account for optional steps.
	if step >= len(steps) {
		return
	}

	if steps[step].Type != expType {
		sstep := cmp.Or(expType.String(), "unknown")
		err = fmt.Errorf("argument #%d of `#%s` %w %s, got %s",
			step, op, ErrPredicateWrongType, sstep, valueFn(steps[step].ValueID))
	}

	return
}
