package sitter

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNewQuery(t *testing.T) {
	t.Parallel()

	//nolint:lll // ok
	testCases := []struct {
		msg, pattern, expStr string
		exp                  error
	}{ // Also add cases for ErrPredicateWrongStart
		{
			"#match?: too few arguments", `((expression) @capture (#match? "this"))`,
			"predicate error: wrong arguments # for #match? (expected 2, got 1) at 1:1",
			ErrPredicateArgsWrongCount,
		},
		{
			"#match?: too many arguments", `((expression) @capture (#match? a b "this"))`,
			"predicate error: wrong arguments # for #match? (expected 2, got 3) at 1:1",
			ErrPredicateArgsWrongCount,
		},
		{
			"#match?: need a capture as first argument", `((expression) @capture (#match? "a" "this"))`,
			`predicate error: invalid type for #match? (arg #1 must be a Capture, got String "a") at 1:1`,
			ErrPredicateWrongType,
		},
		{
			"#match?: need a string as second argument", `((expression) @capture (#match? @capture @capture))`,
			`predicate error: invalid type for #match? (arg #2 must NOT be a Capture, got Capture "@capture") at 1:1`,
			ErrPredicateWrongType,
		},
		{
			"#match?: broken regex", `((expression) @capture (#match? @capture "^[A-Z"))`,
			"predicate error: invalid regex: error parsing regexp: missing closing ]: `[A-Z` at 1:1",
			ErrPredicateRegex,
		},
		{"#match?: success test", `((expression) @capture (#match? @capture "^[A-Z]"))`, "", nil},
		{
			"#not-match?: too few arguments", `((expression) @capture (#not-match? "this"))`,
			"predicate error: wrong arguments # for #not-match? (expected 2, got 1) at 1:1",
			ErrPredicateArgsWrongCount,
		},
		{
			"#not-match?: too many arguments", `((expression) @capture (#not-match? a b "this"))`,
			"predicate error: wrong arguments # for #not-match? (expected 2, got 3) at 1:1",
			ErrPredicateArgsWrongCount,
		},
		{
			"#not-match?: need a capture as first argument", `((expression) @capture (#not-match? "a" "this"))`,
			`predicate error: invalid type for #not-match? (arg #1 must be a Capture, got String "a") at 1:1`,
			ErrPredicateWrongType,
		},
		{
			"#not-match?: need a string as second argument", `((expression) @capture (#not-match? @capture @capture))`,
			`predicate error: invalid type for #not-match? (arg #2 must NOT be a Capture, got Capture "@capture") at 1:1`,
			ErrPredicateWrongType,
		},
		{"#not-match?: success test", `((expression) @capture (#not-match? @capture "^[A-Z]"))`, "", nil},
		{
			"#eq?: too few arguments", `((expression) @capture (#eq? "this"))`,
			"predicate error: wrong arguments # for #eq? (expected 2, got 1) at 1:1",
			ErrPredicateArgsWrongCount,
		},
		{
			"#eq?: too many arguments", `((expression) @capture (#eq? a b "this"))`,
			"predicate error: wrong arguments # for #eq? (expected 2, got 3) at 1:1",
			ErrPredicateArgsWrongCount,
		},
		{
			"#eq?: need a capture as first argument", `((expression) @capture (#eq? "a" "this"))`,
			`predicate error: invalid type for #eq? (arg #1 must be a Capture, got String "a") at 1:1`,
			ErrPredicateWrongType,
		},
		{"#eq?: success test", `((expression) @capture (#eq? @capture "this"))`, "", nil},
		{"#eq?: success double predicate test", `((expression) @capture (#eq? @capture @capture) (#eq? @capture "this"))`, "", nil},
		{"#eq?: success test", `((expression) @capture (#eq? @capture @capture))`, "", nil},
		{
			"#not-eq?: too few arguments", `((expression) @capture (#not-eq? "this"))`,
			"predicate error: wrong arguments # for #not-eq? (expected 2, got 1) at 1:1",
			ErrPredicateArgsWrongCount,
		},
		{
			"#not-eq?: too many arguments", `((expression) @capture (#not-eq? a b "this"))`,
			"predicate error: wrong arguments # for #not-eq? (expected 2, got 3) at 1:1",
			ErrPredicateArgsWrongCount,
		},
		{
			"#not-eq?: need a capture as first argument", `((expression) @capture (#not-eq? "a" "this"))`,
			`predicate error: invalid type for #not-eq? (arg #1 must be a Capture, got String "a") at 1:1`,
			ErrPredicateWrongType,
		},
		{"#not-eq?: success test", `((expression) @capture (#not-eq? @capture "this"))`, "", nil},
		{"#not-eq?: success test", `((expression) @capture (#not-eq? @capture @capture))`, "", nil},
		{
			"#is?: too few arguments", `((expression) @capture (#is?))`,
			"predicate error: wrong arguments # for #is? (expected [1..3], got 0) at 1:1",
			ErrPredicateArgsWrongCount,
		},
		{
			"#is?: too many arguments", `((expression) @capture (#is? a b "this"))`,
			"predicate error: invalid argument for #is? (unexpected argument #3 @this) at 1:1",
			ErrPredicateInvalidArg,
		},
		// {"#is?: need a string as first argument", `((expression) @capture (#is? @capture "this"))`, "", ErrPredicateWrongType},
		// {"#is?: need a string as second argument", `((expression) @capture (#is? "this" @capture))`, "", ErrPredicateWrongType},
		{"#is?: success test", `((expression) @capture (#is? "foo" "bar"))`, "", nil},
		{
			"#is-not?: too few arguments", `((expression) @capture (#is-not?))`,
			"predicate error: wrong arguments # for #is-not? (expected [1..3], got 0) at 1:1",
			ErrPredicateArgsWrongCount,
		},
		{
			"#is-not?: too many arguments", `((expression) @capture (#is-not? a b "this"))`,
			"predicate error: invalid argument for #is-not? (unexpected argument #3 @this) at 1:1",
			ErrPredicateInvalidArg,
		},
		// {"#is-not?: need a string as first argument", `((expression) @capture (#is-not? @capture "this"))`, "", ErrPredicateWrongType},
		// {"#is-not?: need a string as second argument", `((expression) @capture (#is-not? "this" @capture))`, "", ErrPredicateWrongType},
		{"#is-not?: success test", `((expression) @capture (#is-not? "foo" "bar"))`, "", nil},
		{
			"#set!: too few arguments", `((expression) @capture (#set!))`,
			"predicate error: wrong arguments # for #set! (expected [1..3], got 0) at 1:1",
			ErrPredicateArgsWrongCount,
		},
		{
			"#set!: too many arguments", `((expression) @capture (#set! a b "this"))`,
			"predicate error: invalid argument for #set! (unexpected argument #3 @this) at 1:1",
			ErrPredicateInvalidArg,
		},
		// {"#set!: need a string as first argument", `((expression) @capture (#set! @capture "this"))`, "", ErrPredicateWrongType},
		// {"#set!: need a string as second argument", `((expression) @capture (#set! "this" @capture))`, "", ErrPredicateWrongType},
		{"#set!: success test", `((expression) @capture (#set! "foo" "bar"))`, "", nil},
	}

	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			t.Parallel()

			q, err := NewQuery(gr, []byte(tc.pattern))
			success := strings.Contains(tc.msg, "success")

			if (err != nil) && success {
				t.Fatal(tc.msg)
			}

			if !errors.Is(err, tc.exp) {
				t.Fatalf("Expected %v, got %v", tc.exp, err)
			}

			if err != nil && tc.expStr != "" && err.Error() != tc.expStr {
				t.Fatalf("Expected\n%s; got\n%v\n", tc.expStr, err)
			}

			if (q == nil) && success {
				t.Fatal(tc.msg)
			}
		})
	}
}

func TestNewDetailedQueryError(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryErrorString(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryClose(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryPatternCount(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryCaptureCount(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryStringCount(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryStartByteForPattern(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryEndByteForPattern(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryPredicatesForPattern(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryIsPatternRooted(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryIsPatternNonLocal(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryIsPatternGuaranteedAtStep(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryCaptureNameForID(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryCaptureQuantifierForID(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryStringValueForID(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryDisableCapture(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryDisablePattern(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestNewQueryCursor(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNewQueryMatch(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryCursorClose(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryCursorExec(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryCursorDidExceedMatchLimit(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryCursorMatchLimit(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryCursorSetMatchLimit(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryCursorSetTimeout(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryCursorTimeout(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryCursorSetByteRange(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryCursorSetPointRange(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryCursorNextMatch(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryCursorRemoveMatch(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryCursorNextCapture(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryCursorSetMaxStartDepth(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestQueryCursorCopy(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryCursorFilterPredicates(t *testing.T) {
	t.Parallel()

	sumLR := `((sum left: (expression (number) @left) right: (expression (number) @right))`
	testCases := []struct {
		input, query string
		exp          int
	}{
		{`// foo`, `((comment) @capture (#match? @capture "^// [a-z]+$"))`, 1},
		{`// foo123`, `((comment) @capture (#match? @capture "^// [a-z]+$"))`, 0},
		{`// foo`, `((comment) @capture (#not-match? @capture "^// [a-z]+$"))`, 0},
		{`// foo123`, `((comment) @capture (#not-match? @capture "^// [a-z]+$"))`, 1},
		{`// foo`, `((comment) @capture (#eq? @capture "// foo"))`, 1},
		{`// foo`, `((comment) @capture (#eq? @capture "// bar"))`, 0},
		{`// foo`, `((comment) @capture (#eq? @capture "// foo") (#eq? @capture "// bar"))`, 0},
		{`1234 + 1234`, sumLR + ` (#eq? @left @right))`, 2},
		{`1234 + 4321`, sumLR + ` (#eq? @left @right))`, 0},
		{`// foo`, `((comment) @capture (#not-eq? @capture "// foo"))`, 0},
		{`// foo`, `((comment) @capture (#not-eq? @capture "// bar"))`, 1},
		{`1234 + 1234`, sumLR + ` (#not-eq? @left @right))`, 0},
		{`1234 + 4321`, sumLR + ` (#not-eq? @left @right))`, 2},
		{`1234 + 4321`, sumLR + ` (#eq? @left 1234))`, 2},
		{`1234 + 4321`, sumLR + ` (#eq? @left 1234) (#not-eq? @left @right))`, 2},
		{`1234 + 4321`, sumLR + ` (#eq? @left 1234) (#eq? @left 4321))`, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			p := NewParser()
			p.SetLanguage(gr)

			in := []byte(tc.input)

			tree, err := p.ParseString(context.TODO(), nil, in)
			if err != nil {
				t.Fatal("Expected no error, got", err)
			}

			root := tree.RootNode()

			q, err := NewQuery(gr, []byte(tc.query))
			if err != nil {
				t.Fatal("Expected no error, got", err)
			}

			qc := NewQueryCursor()
			matches := qc.Matches(q, root, []byte(tc.input))

			m := matches.Next()

			if tc.exp == 0 && m != nil {
				t.Fatalf("Expected no match, got %d for %v", len(m.Captures), tc.query)
			}

			if m == nil && tc.exp > 0 {
				t.Fatal("Expected a match, got none for", tc.query)
			}

			if tc.exp == 0 {
				return
			}

			if x := len(m.Captures); x != tc.exp {
				t.Fatalf("Expected %d filtered captures, got %d", tc.exp, x)
			}
		})
	}
}

func TestDetailedQueryErrorError(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryPredicateStepsSplit(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryPredicateStepsAssertValid(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryPredicateStepsAssertCount(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestQueryPredicateStepsAssertStepType(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}
