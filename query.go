package sitter

// #include "sitter.h"
import "C"

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"unsafe"
)

// Query is a tree query, compiled from a string of S-expressions.
// The query itself is immutable. The mutable state used in the
// process of executing the query is stored in a `TSQueryCursor`.
type Query struct {
	c *C.TSQuery

	captureNames       []string
	captureQuantifiers [][]CaptureQuantifier
	TextPredicates     [][]TextPredicateCapture
	propertySettings   [][]QueryProperty
	propertyPredicates [][]PropertyPredicate
	generalPredicates  [][]QueryPredicate
}

type Predicator func(_ *Query, _ QueryPredicateSteps, op string, row uint,
	strVal, cptVal func(int) func() string) (any, error)

type CaptureQuantifier = C.TSQuantifier

type TextPredicateCapture struct {
	Value         any
	Type          TextPredicateType
	CaptureID     uint
	Positive      bool
	MatchAllNodes bool
}

type TextPredicateType int

// QueryProperty holds a kv pair associated with a particular pattern in a [Query].
type QueryProperty struct {
	CaptureID *uint
	Value     *string
	Key       string
}

type QueryPredicateArg struct {
	CaptureID *uint
	String    *string
}

type QueryPredicate struct {
	Operator string
	Args     []QueryPredicateArg
}

type PropertyPredicate struct {
	Property QueryProperty
	Positive bool
}

// QueryCursor is a stateful struct used to execute a query on a tree.
type QueryCursor struct {
	c *C.TSQueryCursor
}

// QueryCapture is a captured node by a query with an index.
type QueryCapture struct {
	Node  Node
	Index uint32
}

// QueryMatch allows you to iterate over the matches.
type QueryMatch struct {
	cursor       *QueryCursor
	Captures     []QueryCapture
	PatternIndex uint
	ID           uint
}

// QueryMatches holds a sequence of [QueryMatch]es associated with a given [QueryCursor].
type QueryMatches struct {
	cursor *QueryCursor
	query  *Query
	text   []byte
}

// QueryCaptures holds a sequence of [QueryCapture]s associated with a given [QueryCursor].
type QueryCaptures struct {
	cursor *QueryCursor
	query  *Query
	text   []byte
}

// QueryPredicateStep represents one step in a predicate.
type QueryPredicateStep struct {
	Type    QueryPredicateStepType
	ValueID uint32
}

// QueryPredicateStepType represents type of step in a predicate.
type QueryPredicateStepType uint32

// QueryPredicateSteps holds all the steps for a predicate.
type QueryPredicateSteps []QueryPredicateStep

//nolint:godox // ok
// TODO: DRY a little inner vs Kind in QueryError.

// QueryError holds detailed query error.
// The Offset argument will be set to the byte offset of the error,
// the Kind argument will be set to a value that indicates the type,
// of error and inner will represent the error class.
type QueryError struct {
	inner   error
	Message string
	Offset  uint
	Kind    QueryErrorKind
	Point
}

// QueryErrorKind indicates the type of QueryErrorKind.
type QueryErrorKind uint32

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
	QueryErrorNone      QueryErrorKind = C.TSQueryErrorNone
	QueryErrorSyntax    QueryErrorKind = C.TSQueryErrorSyntax
	QueryErrorNodeType  QueryErrorKind = C.TSQueryErrorNodeType
	QueryErrorField     QueryErrorKind = C.TSQueryErrorField
	QueryErrorCapture   QueryErrorKind = C.TSQueryErrorCapture
	QueryErrorStructure QueryErrorKind = C.TSQueryErrorStructure
	QueryErrorLanguage  QueryErrorKind = C.TSQueryErrorLanguage
	QueryErrorPredicate QueryErrorKind = 100
)

const (
	CaptureQuantifierZero       CaptureQuantifier = C.TSQuantifierZero
	CaptureQuantifierZeroOrOne  CaptureQuantifier = C.TSQuantifierZeroOrOne
	CaptureQuantifierZeroOrMore CaptureQuantifier = C.TSQuantifierZeroOrMore
	CaptureQuantifierOne        CaptureQuantifier = C.TSQuantifierOne
	CaptureQuantifierOneOrMore  CaptureQuantifier = C.TSQuantifierOneOrMore
)

const (
	TextPredicateTypeEqCapture TextPredicateType = iota
	TextPredicateTypeEqString
	TextPredicateTypeMatchString
	TextPredicateTypeAnyString
)

const (
	maxUint16 = uint16(C.UINT16_MAX)
	maxUint32 = uint32(C.UINT32_MAX)
	catchall  = "default"
	// UnlimitedMaxDepth is used for turning off max depth limit for query cursor.
	UnlimitedMaxDepth = maxUint32
)

// Query related errors.
var (
	ErrPredicateBase           = errors.New("predicate error")
	ErrPredicateArgsWrongCount = fmt.Errorf("%w: wrong arguments #", ErrPredicateBase)
	ErrPredicateInvalidArg     = fmt.Errorf("%w: invalid argument", ErrPredicateBase)
	ErrPredicateWrongStart     = fmt.Errorf("%w: must begin with a literal value", ErrPredicateBase)
	ErrPredicateWrongType      = fmt.Errorf("%w: invalid type", ErrPredicateBase)
	ErrPredicateRegex          = fmt.Errorf("%w: invalid regex", ErrPredicateBase)

	ErrPredicateFnBase     = errors.New("predicate fn error")
	ErrPredicateFnWrongRet = fmt.Errorf("%w: invalid return type", ErrPredicateFnBase)
	ErrPredicateFnMissing  = fmt.Errorf("%w: none registered", ErrPredicateFnBase)
)

// TODO: This gives us the opportunity to allow end users to register their
// own predicates.
//
//nolint:godox // ok
var predicators = map[string]Predicator{ //nolint:gochecknoglobals // ok
	"eq?":            assertPredEq,
	"not-eq?":        assertPredEq,
	"any-eq?":        assertPredEq,
	"any-not-eq?":    assertPredEq,
	"match?":         assertPredMatch,
	"not-match?":     assertPredMatch,
	"any-match?":     assertPredMatch,
	"any-not-match?": assertPredMatch,
	"any-of?":        assertPredAny,
	"not-any-of?":    assertPredAny,
	"set!":           assertPredSet,
	"is?":            assertPredIs,
	"is-not?":        assertPredIs,
	catchall:         assertPredDefault,
}

var voidPoint = Point{Row: uint(maxUint32), Column: uint(maxUint32)} //nolint:gochecknoglobals // ok

// NewQuery creates a new query from a string containing one or more S-expression
// patterns. The query is associated with a particular language, and can
// only be run on syntax nodes parsed with that language.
//
// If all of the given patterns are valid, this returns a `TSQuery`.
// If a pattern is invalid, it returns an error which provides two pieces
// of information about the problem:
//  1. The byte offset of the error is written to the `error_offset` parameter.
//  2. The type of error is written to the `error_type` parameter.
//
//nolint:nakedret // ok
func NewQuery(lang *Language, pattern []byte) (q *Query, err error) {
	var (
		errOfs   C.uint32_t
		errType  C.TSQueryError
		bytesPtr *C.char
	)

	if len(pattern) > 0 {
		bytesPtr = (*C.char)(unsafe.Pointer(&pattern[0]))
	}

	c := C.ts_query_new(lang.c(), bytesPtr, C.uint32_t(len(pattern)), &errOfs, &errType)
	if c == nil {
		return nil, newQueryError(lang, pattern, errType, errOfs)
	}

	q = &Query{c: c}
	pc := q.PatternCount()

	q.captureNames = make([]string, 0, q.CaptureCount())
	q.captureQuantifiers = make([][]CaptureQuantifier, 0, pc)
	q.TextPredicates = make([][]TextPredicateCapture, 0, pc)
	q.propertyPredicates = make([][]PropertyPredicate, 0, pc)
	q.propertySettings = make([][]QueryProperty, 0, pc)
	q.generalPredicates = make([][]QueryPredicate, 0, pc)

	q, err = fromRawParts(q, pattern)
	if err != nil {
		return
	}

	runtime.AddCleanup(q, func(c *C.TSQuery) { C.ts_query_delete(c) }, q.c)

	return
}

//nolint:nakedret // ok
func fromRawParts(q *Query, pattern []byte) (_ *Query, err error) { //nolint:funlen,gocognit,cyclop // ok
	// Build a vector of strings to store the capture names.
	for i := range q.CaptureCount() {
		q.captureNames = append(q.captureNames, q.CaptureNameForID(i))
	}

	// Build a vector to store capture qunatifiers.
	for i := range q.PatternCount() {
		cqx := make([]CaptureQuantifier, 0, q.CaptureCount())
		for j := range uint32(cap(cqx)) {
			cqx = append(cqx, q.CaptureQuantifierForID(i, j))
		}

		q.captureQuantifiers = append(q.captureQuantifiers, cqx)
	}

	// Build a vector of strings to represent literal values used in predicates.
	stringValues := make([]string, 0, q.StringCount())
	for i := range uint32(cap(stringValues)) {
		stringValues = append(stringValues, q.StringValueForID(i))
	}

	// Build a vector of strings to represent literal values used in predicates.
	for i := range q.PatternCount() {
		predicateSteps := q.PredicatesForPattern(i)
		byteOffset := q.StartByteForPattern(int(i))
		row := uint(0)

		for i, c := range pattern {
			if i >= int(byteOffset) {
				break
			}

			if c == '\n' {
				row++
			}
		}

		textPredicates := []TextPredicateCapture{}
		propertyPredicates := []PropertyPredicate{}
		propertySettings := []QueryProperty{}
		generalPredicates := []QueryPredicate{}

		for _, steps := range predicateSteps {
			if len(steps) == 0 {
				continue
			}

			if steps[0].Type != QueryPredicateStepTypeString {
				return nil, pErr(ErrPredicateWrongStart, row, "got @"+q.captureNames[steps[0].ValueID])
			}

			strVal, cptVal := func(i int) func() string {
				return func() string { return stringValues[steps[i].ValueID] }
			}, func(i int) func() string {
				return func() string { return "@" + q.captureNames[steps[i].ValueID] }
			}

			op := stringValues[steps[0].ValueID]

			fn := predicators[op]
			if fn == nil {
				fn = predicators[catchall]
			}

			if fn == nil {
				return nil, pErr(ErrPredicateFnMissing, row, op)
			}

			var x any

			x, err = fn(q, steps, op, row, strVal, cptVal)
			if err != nil {
				return
			}

			// Build a predicate for each of the known predicate function names.
			switch v := x.(type) {
			case TextPredicateCapture:
				textPredicates = append(textPredicates, v)
			case PropertyPredicate:
				propertyPredicates = append(propertyPredicates, v)
			case QueryProperty:
				propertySettings = append(propertySettings, v)
			case QueryPredicate:
				generalPredicates = append(generalPredicates, v)
			default:
				return nil, pErr(ErrPredicateFnWrongRet, row,
					fmt.Sprintf("predicator function for %s has an invalid type %T", op, v))
			}
		}

		q.TextPredicates = append(q.TextPredicates, textPredicates)
		q.propertyPredicates = append(q.propertyPredicates, propertyPredicates)
		q.propertySettings = append(q.propertySettings, propertySettings)
		q.generalPredicates = append(q.generalPredicates, generalPredicates)
	}

	return q, nil
}

func (e QueryError) Error() string {
	pre := " for "
	if errors.Is(e.inner, ErrPredicateRegex) {
		pre = ""
	} else if errors.Is(e.inner, ErrPredicateBase) {
		pre = " for #"
	}

	return fmt.Sprintf("%v%s%s at %d:%d", e.inner, pre, e.Message, e.Row+1, e.Column+1)
}

func (e QueryError) Unwrap() error {
	return e.inner
}

func newQueryError(lang *Language, pattern []byte, kind C.TSQueryError, errOfs C.uint) error {
	if QueryErrorKind(kind) == QueryErrorLanguage {
		lErr := LanguageError(lang.Version())
		return &QueryError{Kind: QueryErrorLanguage, inner: error(lErr), Point: voidPoint}
	}

	var (
		offset    = uint(errOfs)
		lineStart uint
		row       uint

		lineContainingError string
	)

	for _, line := range bytes.Split(pattern, []byte("\n")) {
		lineEnd := lineStart + uint(len(line)) + 1
		if lineEnd > offset {
			lineContainingError = string(line)
			break
		}

		lineStart = lineEnd
		row++
	}

	column := offset - lineStart

	var message string

	kkind := QueryErrorKind(kind)

	switch kkind {
	// Error types that report names
	case QueryErrorNodeType, QueryErrorField, QueryErrorCapture:
		suffix := string(pattern[offset:])
		endOffset := len(suffix)

		for i, c := range suffix {
			if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-", c) {
				endOffset = i
				break
			}
		}

		message = suffix[:endOffset]
	default: // Error types that report positions
		message = "Unexpected EOF"
		if lineContainingError != "" {
			message = lineContainingError + "\n" + strings.Repeat(" ", int(offset-lineStart)) + "^"
		}

		kkind = QueryErrorSyntax
		if kkind == QueryErrorStructure {
			kkind = QueryErrorStructure
		}
	}

	return &QueryError{
		Kind:    kkind,
		Point:   Point{Row: row, Column: column},
		Offset:  offset,
		Message: message,
	}
}

// PatternCount returns the number of patterns in the query.
func (q *Query) PatternCount() uint32 {
	return uint32(C.ts_query_pattern_count(q.c))
}

// CaptureNames returns the names of the captures used in the query.
func (q *Query) CaptureNames() []string {
	return q.captureNames
}

// CaptureQuantifiers returns the quantifiers of the captures used in the query.
func (q *Query) CaptureQuantifiers(index uint) []CaptureQuantifier {
	return q.captureQuantifiers[index]
}

// CaptureCount returns the number of captures in the query.
func (q *Query) CaptureCount() uint32 {
	return uint32(C.ts_query_capture_count(q.c))
}

// CaptureIndexForName returns the index for a given capture name.
func (q *Query) CaptureIndexForName(name string) (i int, ok bool) {
	for i, n := range q.captureNames {
		if n == name {
			return i, true
		}
	}

	return
}

// PropertyPredicates returns the properties that are checked for the given pattern index.
//
// This includes predicates with the operators `is?` and `is-not?`.
func (q *Query) PropertyPredicates(index uint) []PropertyPredicate {
	return q.propertyPredicates[index]
}

// PropertySettings returns the properties that are set for the given pattern index.
//
// This includes predicates with the operator `set!`.
func (q *Query) PropertySettings(index uint) []QueryProperty {
	return q.propertySettings[index]
}

// GeneralPredicates returns the other user-defined predicates associated with the given index.
//
// This includes predicate with operators other than:
// * `match?`
// * `eq?` and `not-eq?`
// * `is?` and `is-not?`
// * `set!`
func (q *Query) GeneralPredicates(index uint) []QueryPredicate {
	return q.generalPredicates[index]
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
func (q *Query) StartByteForPattern(i int) uint32 {
	return uint32(C.ts_query_start_byte_for_pattern(q.c, C.uint(i)))
}

// EndByteForPattern returns the byte offset where the given pattern ends
// in the query's source.
//
// This can be useful when combining queries by concatenating their source
// code strings.
func (q *Query) EndByteForPattern(i int) uint32 {
	return uint32(C.ts_query_end_byte_for_pattern(q.c, C.uint(i)))
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
		predicateSteps = append(predicateSteps, QueryPredicateStep{QueryPredicateStepType(stepType), valueID})
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

	C.ts_query_disable_capture(q.c, cName, C.uint(len(name)))
	C.free(unsafe.Pointer(cName))
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
func NewQueryCursor() (qc *QueryCursor) {
	qc = &QueryCursor{c: C.ts_query_cursor_new()}

	runtime.AddCleanup(qc, func(c *C.TSQueryCursor) { C.ts_query_cursor_delete(c) }, qc.c)

	return
}

func newQueryMatch(m *C.TSQueryMatch, cursor *QueryCursor) *QueryMatch {
	var captures []QueryCapture

	if m.capture_count > 0 {
		cCaptures := unsafe.Slice(m.captures, m.capture_count)
		captures = *(*[]QueryCapture)(unsafe.Pointer(&cCaptures))
	}

	return &QueryMatch{
		cursor:       cursor,
		Captures:     captures,
		PatternIndex: uint(m.pattern_index),
		ID:           uint(m.id),
	}
}

// Matches iterates over all of the matches in the order that they were found.
//
// Each match contains the index of the pattern that matched, and a list of
// captures. Because multiple patterns can match the same set of nodes,
// one match may contain captures that appear *before* some of the
// captures from a previous match.
func (qc *QueryCursor) Matches(q *Query, n Node, text []byte) (qm QueryMatches) {
	qc.exec(q, n)
	return QueryMatches{cursor: qc, query: q, text: text}
}

// Captures iterates over all of the individual captures in the order that they
// appear.
//
// This is useful if you don't care about which pattern matched, and just
// want a single, ordered sequence of captures.
func (qc *QueryCursor) Captures(q *Query, n Node, text []byte) QueryCaptures {
	qc.exec(q, n)
	return QueryCaptures{cursor: qc, query: q, text: text}
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

// SetTimeout sets the maximum duration in microseconds that query execution should be allowed to
// take before halting.
//
// If query execution takes longer than this, it will halt early, returning NULL.
// See [`ts_query_cursor_next_match`] or [`ts_query_cursor_next_capture`] for more information.
func (c *QueryCursor) SetTimeout(micros int) {
	C.ts_query_cursor_set_timeout_micros(c.c, C.uint64_t(micros))
}

// Timeout returns the duration in microseconds that query execution is allowed to take.
//
// This is set via [`ts_query_cursor_set_timeout_micros`].
func (c *QueryCursor) Timeout() (micros int) {
	return int(C.ts_query_cursor_timeout_micros(c.c))
}

// SetByteRange sets the range of bytes in which the query will be executed.
func (c *QueryCursor) SetByteRange(start, end uint32) {
	C.ts_query_cursor_set_byte_range(c.c, C.uint(start), C.uint(end))
}

// SetPointRange sets the range of row/column positions in which the query will be executed.
func (c *QueryCursor) SetPointRange(start, end Point) {
	C.ts_query_cursor_set_point_range(c.c, start.c(), end.c())
}

func (c *QueryCursor) NextMatch() (_ *QueryMatch) {
	m := (*C.TSQueryMatch)(C.malloc(C.sizeof_TSQueryMatch))

	defer C.free(unsafe.Pointer(m))

	if C.ts_query_cursor_next_match(c.c, m) {
		return newQueryMatch(m, c)
	}

	return
}

func (c *QueryCursor) NextCapture() (_ *QueryMatch, i uint) {
	m := (*C.TSQueryMatch)(C.malloc(C.sizeof_TSQueryMatch))

	defer C.free(unsafe.Pointer(m))

	var captureIndex C.uint32_t

	if C.ts_query_cursor_next_capture(c.c, m, &captureIndex) {
		return newQueryMatch(m, c), uint(captureIndex)
	}

	return
}

func (c *QueryCursor) RemoveMatch(matchID uint) {
	C.ts_query_cursor_remove_match(c.c, C.uint32_t(matchID))
}

func (qm *QueryMatch) Remove() {
	qm.cursor.RemoveMatch(qm.ID)
}

func (qm *QueryMatch) NodesForCaptureIndex(captureIndex uint) []Node {
	nodes := []Node{}

	for _, capture := range qm.Captures {
		if uint(capture.Index) == captureIndex {
			nodes = append(nodes, capture.Node)
		}
	}

	return nodes
}

func (qm *QueryMatch) satisfiesTextPredicate(q *Query, text []byte) (ok bool) { //nolint:funlen,gocognit,cyclop,lll // ok
	condition := func(predicate TextPredicateCapture) bool {
		switch predicate.Type {
		case TextPredicateTypeEqCapture:
			i := predicate.CaptureID
			j := predicate.Value.(uint) //nolint:errcheck,forcetypeassert // TODO
			nodes1 := qm.NodesForCaptureIndex(i)
			nodes2 := qm.NodesForCaptureIndex(j)

			for len(nodes1) > 0 && len(nodes2) > 0 {
				node1 := nodes1[0]
				node2 := nodes2[0]

				isPositiveMatch := bytes.Equal(text[node1.StartByte():node1.EndByte()], text[node2.StartByte():node2.EndByte()])
				if isPositiveMatch != predicate.Positive && predicate.MatchAllNodes {
					return false
				}

				if isPositiveMatch == predicate.Positive && !predicate.MatchAllNodes {
					return true
				}

				nodes1 = nodes1[1:]
				nodes2 = nodes2[1:]
			}

			return len(nodes1) == 0 && len(nodes2) == 0
		case TextPredicateTypeEqString:
			i := predicate.CaptureID
			s := predicate.Value.(string) //nolint:errcheck,forcetypeassert // TODO

			nodes := qm.NodesForCaptureIndex(i)
			for _, node := range nodes {
				nodeText := text[node.StartByte():node.EndByte()]

				isPositiveMatch := bytes.Equal(nodeText, []byte(s))
				if isPositiveMatch != predicate.Positive && predicate.MatchAllNodes {
					return false
				}

				if isPositiveMatch == predicate.Positive && !predicate.MatchAllNodes {
					return true
				}
			}

			return true
		case TextPredicateTypeMatchString:
			i := predicate.CaptureID
			r := predicate.Value.(*regexp.Regexp) //nolint:errcheck,forcetypeassert // TODO

			nodes := qm.NodesForCaptureIndex(i)
			for _, node := range nodes {
				nodeText := text[node.StartByte():node.EndByte()]

				isPositiveMatch := r.Match(nodeText)
				if isPositiveMatch != predicate.Positive && predicate.MatchAllNodes {
					return false
				}

				if isPositiveMatch == predicate.Positive && !predicate.MatchAllNodes {
					return true
				}
			}

			return true
		case TextPredicateTypeAnyString:
			i := predicate.CaptureID
			v := predicate.Value.([]string) //nolint:errcheck,forcetypeassert // TODO

			nodes := qm.NodesForCaptureIndex(i)
			for _, node := range nodes {
				nodeText := text[node.StartByte():node.EndByte()]
				isPositiveMatch := false

				for _, s := range v {
					if bytes.Equal(nodeText, []byte(s)) {
						isPositiveMatch = true
						break
					}
				}

				if isPositiveMatch != predicate.Positive {
					return false
				}
			}

			return true
		}

		return false
	}

	for _, predicate := range q.TextPredicates[qm.PatternIndex] {
		if !condition(predicate) {
			return false
		}
	}

	return true
}

func NewQueryProperty(key string, value *string, captureID *uint) QueryProperty {
	return QueryProperty{
		Key:       key,
		Value:     value,
		CaptureID: captureID,
	}
}

// Next will return the next match in the sequence of matches.
//
// Subsequent calls to [QueryMatches.Next] will overwrite the memory at the
// same location as prior matches, since the memory is reused. You can think
// of this as a stateful iterator.
// If you need to keep the data of a prior match without it being overwritten,
// you should copy what you need before calling [QueryMatches.Next] again.
//
// If there are no more matches, it will return nil.
func (qm *QueryMatches) Next() *QueryMatch {
	for {
		if result := qm.cursor.NextMatch(); result != nil {
			if result.satisfiesTextPredicate(qm.query, qm.text) {
				return result
			}
		} else {
			return nil
		}
	}
}

// Next will return the next match in the sequence of matches, as well as the
// index of the capture.
//
// Subsequent calls to [QueryCaptures.Next] will overwrite the memory at the
// same location as prior matches, since the memory is reused. You can think
// of this as a stateful iterator.
// If you need to keep the data of a prior match without it being overwritten,
// you should copy what you need before calling [QueryCaptures.Next] again.
//
// If there are no more matches, it will return nil.
func (qc *QueryCaptures) Next() (m *QueryMatch, index uint) {
	for {
		if m, index = qc.cursor.NextCapture(); m != nil {
			if m.satisfiesTextPredicate(qc.query, qc.text) {
				return
			}

			m.Remove()
		} else {
			return
		}
	}
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

// exec executes the query on a given syntax node.
func (c *QueryCursor) exec(q *Query, n Node) {
	x := c.c
	y := q.c
	z := n.c
	// C.ts_query_cursor_exec(c.c, q.c, n.c)
	C.ts_query_cursor_exec(x, y, z)
}

// Non API.

func (steps QueryPredicateSteps) split() (out []QueryPredicateSteps) {
	var curr QueryPredicateSteps

	for _, step := range steps {
		if step.Type == QueryPredicateStepTypeDone {
			out = append(out, curr)
			curr = nil
		} else {
			curr = append(curr, step)
		}
	}

	if len(curr) > 0 {
		out = append(out, curr)
	}

	return
}

// Checks the step count against the given limits.
//
// If only ext is provided it checks for exact step count, f opts[0] is provided
// it checks for an interval,  and if exp is negative it checks the have at least
// -exp steps.
func (q *Query) assertStepCount(op string, row uint, exp, act int, opts ...int) error {
	var exp2 *int

	if len(opts) > 0 {
		exp2 = &opts[0]
	}

	switch {
	case exp2 != nil:
		if act < exp || act > *exp2 {
			return pErr(ErrPredicateArgsWrongCount, row,
				fmt.Sprintf("%s (expected [%d..%d], got %d)", op, exp, *exp2, act))
		}
	case exp < 0:
		if exp = -exp; act < exp {
			return pErr(ErrPredicateArgsWrongCount, row,
				fmt.Sprintf("%s (expected at least %d, got %d)", op, exp, act))
		}
	default:
		if act != exp {
			return pErr(ErrPredicateArgsWrongCount, row,
				fmt.Sprintf("%s (expected %d, got %d)", op, exp, act))
		}
	}

	return nil
}

//nolint:varnamelen // ok
func (q *Query) assertStepType(op string, valFn func() string, argNo int, row uint, actualType, acceptedType int) error { //nolint:lll // ok
	var a, b QueryPredicateStepType

	a = QueryPredicateStepType(actualType)
	b = QueryPredicateStepType(acceptedType)

	failCond := actualType != acceptedType
	be := "be"

	if acceptedType < 0 {
		failCond = actualType == -acceptedType
		b = QueryPredicateStepType(-acceptedType)
		be = "NOT be"
	}

	if failCond {
		return pErr(ErrPredicateWrongType, row,
			fmt.Sprintf("%s (arg #%d must %s a %v, got %v %q)", op, argNo, be, b, a, valFn()))
	}

	return nil
}

func assertPredEq(q *Query, steps QueryPredicateSteps, op string, row uint, strVal, _ func(int) func() string) (_ any, err error) { //nolint:lll // ok
	if err = q.assertStepCount(op, row, 2, len(steps)-1); err != nil { //nolint:mnd // ok
		return
	}

	if err = q.assertStepType(op, strVal(1), 1, row, int(steps[1].Type), int(QueryPredicateStepTypeCapture)); err != nil { //nolint:lll // ok
		return
	}

	isPositive := op == "eq?" || op == "any-eq?"
	matchAll := op == "eq?" || op == "not-eq?"

	if steps[2].Type == QueryPredicateStepTypeCapture {
		return TextPredicateCapture{
			Type:          TextPredicateTypeEqCapture,
			CaptureID:     uint(steps[1].ValueID),
			Value:         uint(steps[2].ValueID),
			Positive:      isPositive,
			MatchAllNodes: matchAll,
		}, nil
	} else {
		return TextPredicateCapture{
			Type:          TextPredicateTypeEqString,
			CaptureID:     uint(steps[1].ValueID),
			Value:         strVal(2)(), //nolint:mnd // ok
			Positive:      isPositive,
			MatchAllNodes: matchAll,
		}, nil
	}
}

func assertPredMatch(q *Query, steps QueryPredicateSteps, op string, row uint, strVal, cptVal func(int) func() string) (_ any, err error) { //nolint:lll // ok
	if err = q.assertStepCount(op, row, 2, len(steps)-1); err != nil { //nolint:mnd // false positive
		return
	}

	if err = q.assertStepType(op, strVal(1), 1, row, int(steps[1].Type), int(QueryPredicateStepTypeCapture)); err != nil { //nolint:lll // ok
		return
	}

	if err = q.assertStepType(op, cptVal(2), 2, row, int(steps[2].Type), -int(QueryPredicateStepTypeCapture)); err != nil { //nolint:lll,mnd // ok
		return
	}

	var regex *regexp.Regexp

	isPositive := op == "match?" || op == "any-match?"
	matchAll := op == "match?" || op == "not-match?"

	regex, err = regexp.Compile(strVal(2)()) //nolint:mnd // ok
	if err != nil {
		return TextPredicateCapture{}, pErr(fmt.Errorf("%w: %w", ErrPredicateRegex, err), row, "")
	}

	return TextPredicateCapture{
		Type:          TextPredicateTypeMatchString,
		CaptureID:     uint(steps[1].ValueID),
		Value:         regex,
		Positive:      isPositive,
		MatchAllNodes: matchAll,
	}, nil
}

func assertPredAny(q *Query, steps QueryPredicateSteps, op string, row uint, strVal, _ func(int) func() string) (_ any, err error) { //nolint:lll // ok
	if err = q.assertStepCount(op, row, -2, len(steps)-1); err != nil {
		return
	}

	if err = q.assertStepType(op, strVal(1), 1, row, int(steps[1].Type), int(QueryPredicateStepTypeCapture)); err != nil { //nolint:lll // ok
		return
	}

	isPositive := op == "any-of?"
	values := []string{}

	for i, arg := range steps[2:] {
		if arg.Type == QueryPredicateStepTypeCapture {
			return TextPredicateCapture{}, pErr(ErrPredicateWrongType, row,
				"#any-of? predicate must be literals, got capture @"+q.captureNames[arg.ValueID])
		}

		values = append(values, strVal(i+2)()) //nolint:mnd // ok
	}

	return TextPredicateCapture{
		Type:          TextPredicateTypeAnyString,
		CaptureID:     uint(steps[1].ValueID),
		Value:         values,
		Positive:      isPositive,
		MatchAllNodes: true,
	}, nil
}

func assertPredSet(q *Query, steps QueryPredicateSteps, op string, row uint, strVal, _ func(int) func() string) (_ any, err error) { //nolint:lll // ok
	if err = q.assertStepCount(op, row, 1, len(steps)-1, 3); err != nil { //nolint:mnd // ok
		return
	}

	return parseProperty(steps, row, op, q.captureNames, strVal)
}

func assertPredIs(q *Query, steps QueryPredicateSteps, op string, row uint, strVal, _ func(int) func() string) (_ any, err error) { //nolint:lll // ok
	ret, err := assertPredSet(q, steps, op, row, strVal, nil)
	if err != nil {
		return
	}

	return PropertyPredicate{
		Property: ret.(QueryProperty), //nolint:errcheck,forcetypeassert // yeah, we know exactly what that is
		Positive: op == "is?",
	}, nil
}

func assertPredDefault(q *Query, steps QueryPredicateSteps, op string, row uint, strVal, _ func(int) func() string) (_ any, err error) { //nolint:lll // ok
	var args []QueryPredicateArg

	for i, a := range steps[1:] {
		if a.Type == QueryPredicateStepTypeCapture {
			args = append(args, QueryPredicateArg{CaptureID: new(uint), String: nil})
			*args[len(args)-1].CaptureID = uint(a.ValueID)
		} else {
			args = append(args, QueryPredicateArg{CaptureID: nil, String: new(string)})
			*args[len(args)-1].String = strVal(i + 1)()
		}
	}

	return QueryPredicate{
		Operator: op,
		Args:     args,
	}, nil
}

func parseProperty(steps QueryPredicateSteps, row uint, functionName string, captureNames []string, strVal func(int) func() string) (_ any, err error) { //nolint:lll // ok
	var (
		captureID *uint
		key       *string
		value     *string
	)

	for i, step := range steps[1:] { //nolint:varnamelen // no, i isn't too short
		switch {
		case step.Type == QueryPredicateStepTypeCapture:
			if captureID != nil {
				return QueryProperty{}, pErr(ErrPredicateInvalidArg, row,
					fmt.Sprintf("%s (unexpected capture #2 name @%s)",
						functionName, captureNames[step.ValueID]))
			}

			captureID = new(uint)
			*captureID = uint(step.ValueID)
		case key == nil:
			k := strVal(i + 1)()
			key = &k
		case value == nil:
			v := strVal(i + 1)()
			value = &v
		default:
			return QueryProperty{}, pErr(ErrPredicateInvalidArg, row,
				fmt.Sprintf("%s (unexpected argument #3 @%s)",
					functionName, strVal(i+1)()))
		}
	}

	if key == nil {
		return QueryProperty{}, pErr(ErrPredicateInvalidArg, row,
			functionName+" (missing key argument)")
	}

	return QueryProperty{Key: *key, Value: value, CaptureID: captureID}, nil
}

func pErr(err error, row uint, msg string) *QueryError {
	return &QueryError{inner: err, Kind: QueryErrorPredicate, Point: Point{Row: row}, Message: msg}
}
