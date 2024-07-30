package sitter //nolint:gocritic // ok

// #include "sitter.h"
import "C" //nolint:gocritic // ok

import (
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"unsafe" //nolint:gocritic // ok
)

// Query API
type Query struct {
	c        *C.TSQuery
	isClosed bool
}

// QueryCursor carries the state needed for processing the queries.
type QueryCursor struct {
	c *C.TSQueryCursor
	t *Tree
	// NOTE: Keep a pointer to the query to avoid garbage collection.
	q *Query

	isClosed bool
}

// QueryCapture is a captured node by a query with an index.
type QueryCapture struct {
	Node  *Node
	Index uint32
}

// QueryMatch - you can then iterate over the matches.
type QueryMatch struct {
	Captures     []QueryCapture
	ID           uint32
	PatternIndex uint16
}

type QueryPredicateStepType int //nolint:revive // TODO

type QueryPredicateStep struct { //nolint:revive // TODO
	Type    QueryPredicateStepType
	ValueID uint32
}

type Quantifier int //nolint:revive // TODO

// QueryErrorType indicates the type of QueryError.
type QueryErrorType int

// QueryError - if there is an error in the query,
// then the Offset argument will be set to the byte offset of the error,
// and the Type argument will be set to a value that indicates the type of error.
type QueryError struct {
	Message string
	Type    QueryErrorType
	Offset  uint32
}

// Possible query predicate steps.
const (
	QueryPredicateStepTypeDone QueryPredicateStepType = iota
	QueryPredicateStepTypeCapture
	QueryPredicateStepTypeString
)

// Possible quantifiers.
const (
	QuantifierZero = iota
	QuantifierZeroOrOne
	QuantifierZeroOrMore
	QuantifierOne
	QuantifierOneOrMore
)

// Error types.
const (
	QueryErrorNone QueryErrorType = iota
	QueryErrorSyntax
	QueryErrorNodeType
	QueryErrorField
	QueryErrorCapture
	QueryErrorStructure
	QueryErrorLanguage
)

// UnlimitedMaxDepth is used for turning off max depth limit for query cursor.
const UnlimitedMaxDepth = uint32(C.UINT32_MAX)

// NewQuery creates a new query from a string containing one or more S-expression
// patterns. The query is associated with a particular language, and can
// only be run on syntax nodes parsed with that language.
//
// If all of the given patterns are valid, this returns a `TSQuery`.
// If a pattern is invalid, it returns an error which provides two pieces
// of information about the problem:
//  1. The byte offset of the error is written to the `error_offset` parameter.
//  2. The type of error is written to the `error_type` parameter.
func NewQuery(pattern []byte, lang *Language) (*Query, error) { //nolint:funlen,gocognit // ok
	var (
		erroff  C.uint32_t
		errtype C.TSQueryError
	)

	input := C.CBytes(pattern)
	c := C.ts_query_new( //nolint:varnamelen // ok
		(*C.struct_TSLanguage)(lang.ptr),
		(*C.char)(input),
		C.uint32_t(len(pattern)),
		&erroff,
		&errtype, //nolint:nlreturn // false positive
	)

	C.free(input)

	if errtype != C.TSQueryError(QueryErrorNone) {
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
		errorType := QueryErrorType(errtype)
		errorTypeToString := QueryErrorTypeToString(errorType)

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
					errorTypeToString, m[0], line, column)
			} else {
				message = fmt.Sprintf("invalid %s at line %d column %d",
					errorTypeToString, line, column)
			}

		// Errors the report position: QueryErrorSyntax, QueryErrorStructure, QueryErrorLanguage.
		default:
			s := string(pattern[errorOffset:])
			lines := strings.Split(s, "\n")
			whitespace := strings.Repeat(" ", column)
			message = fmt.Sprintf("invalid %s at line %d column %d\n%s\n%s^",
				errorTypeToString, line, column,
				lines[0], whitespace)
		}

		return nil, &QueryError{
			Offset:  errorOffset,
			Type:    errorType,
			Message: message,
		}
	}

	q := &Query{c: c}

	// Copied from: https://github.com/klothoplatform/go-tree-sitter/commit/e351b20167b26d515627a4a1a884528ede5fef79
	// this is just used for syntax validation - it does not actually filter anything
	for i := range q.PatternCount() {
		predicates := q.PredicatesForPattern(i)

		for _, steps := range predicates {
			if len(steps) == 0 {
				continue
			}

			if steps[0].Type != QueryPredicateStepTypeString {
				return nil, errors.New("predicate must begin with a literal value")
			}

			switch operator := q.StringValueForID(steps[0].ValueID); operator {
			case "eq?", "not-eq?":
				if len(steps) != 4 {
					return nil, fmt.Errorf("wrong number of arguments to `#%s` predicate. Expected 2, got %d", operator, len(steps)-2)
				}

				if steps[1].Type != QueryPredicateStepTypeCapture {
					return nil, fmt.Errorf("first argument of `#%s` predicate must be a capture. Got %s", operator, q.StringValueForID(steps[1].ValueID))
				}
			case "match?", "not-match?":
				if len(steps) != 4 {
					return nil, fmt.Errorf("wrong number of arguments to `#%s` predicate. Expected 2, got %d", operator, len(steps)-2)
				}

				if steps[1].Type != QueryPredicateStepTypeCapture {
					return nil, fmt.Errorf("first argument of `#%s` predicate must be a capture. Got %s", operator, q.StringValueForID(steps[1].ValueID))
				}

				if steps[2].Type != QueryPredicateStepTypeString {
					return nil, fmt.Errorf("second argument of `#%s` predicate must be a string. Got %s", operator, q.StringValueForID(steps[2].ValueID))
				}
			case "set!", "is?", "is-not?":
				if len(steps) < 3 || len(steps) > 4 {
					return nil, fmt.Errorf("wrong number of arguments to `#%s` predicate. Expected 1 or 2, got %d", operator, len(steps)-2)
				}

				if steps[1].Type != QueryPredicateStepTypeString {
					return nil, fmt.Errorf("first argument of `#%s` predicate must be a string. Got %s", operator, q.StringValueForID(steps[1].ValueID))
				}

				if len(steps) > 2 && steps[2].Type != QueryPredicateStepTypeString {
					return nil, fmt.Errorf("second argument of `#%s` predicate must be a string. Got %s", operator, q.StringValueForID(steps[2].ValueID))
				}
			}
		}
	}

	runtime.SetFinalizer(q, (*Query).Close)

	return q, nil
}

// QueryErrorTypeToString converts a query error to string.
func QueryErrorTypeToString(errorType QueryErrorType) string {
	switch errorType {
	case QueryErrorNone:
		return "none"
	case QueryErrorNodeType:
		return "node type"
	case QueryErrorField:
		return "field"
	case QueryErrorCapture:
		return "capture"
	case QueryErrorSyntax:
		return "syntax"
	default:
		return "unknown"
	}
}

// Close should be called to ensure that all the memory used by the query is freed.
//
// As the constructor in go-tree-sitter would set this func call through runtime.SetFinalizer,
// parser.Close() will be called by Go's garbage collector and users would not have to call this manually.
func (q *Query) Close() {
	if !q.isClosed {
		C.ts_query_delete(q.c)
	}

	q.isClosed = true
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

/** TODO
 * Get the byte offset where the given pattern starts in the query's source.
 *
 * This can be useful when combining queries by concatenating their source
 * code strings.
 /
uint32_t ts_query_start_byte_for_pattern(const TSQuery *self, uint32_t pattern_index);
*/

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
func (q *Query) PredicatesForPattern(patternIndex uint32) [][]QueryPredicateStep {
	var ( //nolint:prealloc // no
		length         C.uint32_t
		predicateSteps []QueryPredicateStep
	)

	cPredicateStep := C.ts_query_predicates_for_pattern(q.c, C.uint32_t(patternIndex), &length)
	cPredicateSteps := unsafe.Slice(cPredicateStep, int(length))

	for _, s := range cPredicateSteps {
		stepType := QueryPredicateStepType(s._type)
		valueID := uint32(s.value_id)
		predicateSteps = append(predicateSteps, QueryPredicateStep{stepType, valueID})
	}

	return splitPredicates(predicateSteps)
}

/* TODO
 * Check if the given pattern in the query has a single root node.
 /
bool ts_query_is_pattern_rooted(const TSQuery *self, uint32_t pattern_index);

/* TODO
 * Check if the given pattern in the query is 'non local'.
 *
 * A non-local pattern has multiple root nodes and can match within a
 * repeating sequence of nodes, as specified by the grammar. Non-local
 * patterns disable certain optimizations that would otherwise be possible
 * when executing a query on a specific range of a syntax tree.
 /
bool ts_query_is_pattern_non_local(const TSQuery *self, uint32_t pattern_index);

/* TODO
 * Check if a given pattern is guaranteed to match once a given step is reached.
 * The step is specified by its byte offset in the query's source code.
 /
bool ts_query_is_pattern_guaranteed_at_step(const TSQuery *self, uint32_t byte_offset);
*/

// CaptureNameForID returns the name and length of one of the query's captures,
// or one of the  query's string literals. Each capture and string is associated
// with a  numeric id based on the order that it appeared in the query's source.
func (q *Query) CaptureNameForID(id uint32) string {
	var length C.uint32_t

	name := C.ts_query_capture_name_for_id(q.c, C.uint32_t(id), &length)

	return C.GoStringN(name, C.int(length))
}

// CaptureQuantifierForID returns the quantifier of the query's captures.
// Each capture is associated with a numeric id based on the order that it
// appeared in the query's source.
func (q *Query) CaptureQuantifierForID(id, captureID uint32) Quantifier {
	return Quantifier(C.ts_query_capture_quantifier_for_id(q.c, C.uint32_t(id), C.uint32_t(captureID)))
}

// StringValueForID returns the string value associated with the given query id.
func (q *Query) StringValueForID(id uint32) string {
	var length C.uint32_t

	value := C.ts_query_string_value_for_id(q.c, C.uint32_t(id), &length)

	return C.GoStringN(value, C.int(length))
}

/** TODO
 * Disable a certain capture within a query.
 *
 * This prevents the capture from being returned in matches, and also avoids
 * any resource usage associated with recording the capture. Currently, there
 * is no way to undo this.
 /
void ts_query_disable_capture(TSQuery *self, const char *name, uint32_t length);

/** TODO
 * Disable a certain pattern within a query.
 *
 * This prevents the pattern from matching and removes most of the overhead
 * associated with the pattern. Currently, there is no way to undo this.
 /
void ts_query_disable_pattern(TSQuery *self, uint32_t pattern_index);
*/

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

	runtime.SetFinalizer(qc, (*QueryCursor).Close)

	return qc
}

// Close should be called to ensure that all the memory used by the query cursor is freed.
//
// As the constructor in go-tree-sitter would set this func call through runtime.SetFinalizer,
// parser.Close() will be called by Go's garbage collector and users would not have to call this manually.
func (qc *QueryCursor) Close() {
	if !qc.isClosed {
		C.ts_query_cursor_delete(qc.c)
	}

	qc.isClosed = true
}

// Exec executes the query on a given syntax node.
func (qc *QueryCursor) Exec(q *Query, n *Node) {
	qc.q = q
	qc.t = n.t

	C.ts_query_cursor_exec(qc.c, q.c, n.c)
}

/** TODO
 * Manage the maximum number of in-progress matches allowed by this query
 * cursor.
 *
 * Query cursors have an optional maximum capacity for storing lists of
 * in-progress captures. If this capacity is exceeded, then the
 * earliest-starting match will silently be dropped to make room for further
 * matches. This maximum capacity is optional — by default, query cursors allow
 * any number of pending matches, dynamically allocating new space for them as
 * needed as the query is executed.
bool ts_query_cursor_did_exceed_match_limit(const TSQueryCursor *self);
uint32_t ts_query_cursor_match_limit(const TSQueryCursor *self);
void ts_query_cursor_set_match_limit(TSQueryCursor *self, uint32_t limit);
*/

// SetByteRange sets the range of bytes in which the query will be executed.
func (qc *QueryCursor) SetByteRange(start, end uint32) {
	C.ts_query_cursor_set_byte_range(qc.c, C.uint32_t(start), C.uint32_t(end))
}

// SetPointRange sets the range of row/column positions in which the query will be executed.
func (qc *QueryCursor) SetPointRange(startPoint, endPoint Point) {
	cStartPoint := C.TSPoint{
		row:    C.uint32_t(startPoint.Row),
		column: C.uint32_t(startPoint.Column),
	}
	cEndPoint := C.TSPoint{
		row:    C.uint32_t(endPoint.Row),
		column: C.uint32_t(endPoint.Column),
	}

	C.ts_query_cursor_set_point_range(qc.c, cStartPoint, cEndPoint)
}

// NextMatch iterates over matches.
// This function will return (nil, false) when there are no more matches.
// Otherwise, it will populate the QueryMatch with data
// about which pattern matched and which nodes were captured.
func (qc *QueryCursor) NextMatch() (*QueryMatch, bool) {
	var cqm C.TSQueryMatch

	if ok := C.ts_query_cursor_next_match(qc.c, &cqm); !bool(ok) { //nolint:gocritic // ok
		return nil, false
	}

	qm := &QueryMatch{
		ID:           uint32(cqm.id),
		PatternIndex: uint16(cqm.pattern_index),
	}

	cqc := unsafe.Slice(cqm.captures, int(cqm.capture_count))
	for _, c := range cqc {
		idx := uint32(c.index)
		node := qc.t.cachedNode(c.node)
		qm.Captures = append(qm.Captures, QueryCapture{Index: idx, Node: node})
	}

	return qm, true
}

// RemoveMatch does smth... TODO
func (qc *QueryCursor) RemoveMatch(matchID uint32) {
	C.ts_query_cursor_remove_match(qc.c, C.uint32_t(matchID))
}

// NextCapture advances to the next capture of the currently running query.
//
// If there is a capture, write its match to `*match` and its index within
// the matche's capture list to `*capture_index`. Otherwise, return `false`.
func (qc *QueryCursor) NextCapture() (*QueryMatch, uint32, bool) {
	var (
		cqm          C.TSQueryMatch
		captureIndex C.uint32_t
	)

	if ok := C.ts_query_cursor_next_capture(qc.c, &cqm, &captureIndex); !bool(ok) { //nolint:gocritic // ok
		return nil, 0, false
	}

	qm := &QueryMatch{
		ID:           uint32(cqm.id),
		PatternIndex: uint16(cqm.pattern_index),
	}

	cqc := unsafe.Slice(cqm.captures, int(cqm.capture_count))
	for _, c := range cqc {
		idx := uint32(c.index)
		node := qc.t.cachedNode(c.node)
		qm.Captures = append(qm.Captures, QueryCapture{Index: idx, Node: node})
	}

	return qm, uint32(captureIndex), true
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
func (qc *QueryCursor) SetMaxStartDepth(maxStartDepth uint32) {
	C.ts_query_cursor_set_max_start_depth(qc.c, C.uint32_t(maxStartDepth))
}

// Non API.

// FilterPredicates filters the given query match with the applicable predicates.
func (qc *QueryCursor) FilterPredicates(m *QueryMatch, input []byte) *QueryMatch { //nolint:funlen,gocognit // ok
	qm := &QueryMatch{
		ID:           m.ID,
		PatternIndex: m.PatternIndex,
	}

	q := qc.q

	predicates := q.PredicatesForPattern(uint32(qm.PatternIndex))
	if len(predicates) == 0 {
		qm.Captures = m.Captures
		return qm
	}

	// track if we matched all predicates globally
	matchedAll := true

	// check each predicate against the match
	for _, steps := range predicates {
		operator := q.StringValueForID(steps[0].ValueID)

		switch operator {
		case "eq?", "not-eq?":
			isPositive := operator == "eq?"

			expectedCaptureNameLeft := q.CaptureNameForID(steps[1].ValueID)

			if steps[2].Type == QueryPredicateStepTypeCapture {
				expectedCaptureNameRight := q.CaptureNameForID(steps[2].ValueID)

				var nodeLeft, nodeRight *Node

				for _, c := range m.Captures {
					captureName := q.CaptureNameForID(c.Index)

					if captureName == expectedCaptureNameLeft {
						nodeLeft = c.Node
					}

					if captureName == expectedCaptureNameRight {
						nodeRight = c.Node
					}

					if nodeLeft != nil && nodeRight != nil {
						if (nodeLeft.Content(input) == nodeRight.Content(input)) != isPositive {
							matchedAll = false
						}

						break
					}
				}
			} else {
				expectedValueRight := q.StringValueForID(steps[2].ValueID)

				for _, c := range m.Captures {
					captureName := q.CaptureNameForID(c.Index)

					if expectedCaptureNameLeft != captureName {
						continue
					}

					if (c.Node.Content(input) == expectedValueRight) != isPositive {
						matchedAll = false

						break
					}
				}
			}

			if !matchedAll {
				break //nolint:staticcheck // TODO: This is an ineffective break statement. Is it a bug or just superfluous?
			}
		case "match?", "not-match?":
			isPositive := operator == "match?"

			expectedCaptureName := q.CaptureNameForID(steps[1].ValueID)
			regex := regexp.MustCompile(q.StringValueForID(steps[2].ValueID))

			for _, c := range m.Captures {
				captureName := q.CaptureNameForID(c.Index)
				if expectedCaptureName != captureName {
					continue
				}

				if regex.MatchString(c.Node.Content(input)) != isPositive {
					matchedAll = false
					break
				}
			}
		}
	}

	if matchedAll {
		qm.Captures = append(qm.Captures, m.Captures...)
	}

	return qm
}

func (qe *QueryError) Error() string {
	return qe.Message
}

// Copied From: https://github.com/klothoplatform/go-tree-sitter/commit/e351b20167b26d515627a4a1a884528ede5fef79

func splitPredicates(steps []QueryPredicateStep) [][]QueryPredicateStep {
	var (
		predicateSteps [][]QueryPredicateStep
		currentSteps   = make([]QueryPredicateStep, 0, len(steps))
	)

	for _, step := range steps {
		currentSteps = append(currentSteps, step)
		if step.Type == QueryPredicateStepTypeDone {
			predicateSteps = append(predicateSteps, currentSteps)
			currentSteps = []QueryPredicateStep{}
		}
	}

	return predicateSteps
}
