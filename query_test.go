package sitter

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestQueryWithPredicates(t *testing.T) {
	t.Parallel()

	//nolint:lll // ok
	testCases := []struct {
		msg, pattern string
		exp          error
	}{ // Also add cases for ErrPredicateWrongStart
		{"#match?: too few arguments", `((expression) @capture (#match? "this"))`, ErrPredicateArgsWrongCount},
		{"#match?: too many arguments", `((expression) @capture (#match? a b "this"))`, ErrPredicateArgsWrongCount},
		{"#match?: need a capture as first argument", `((expression) @capture (#match? "a" "this"))`, ErrPredicateWrongType},
		{"#match?: need a string as second argument", `((expression) @capture (#match? @capture @capture))`, ErrPredicateWrongType},
		{"#match?: success test", `((expression) @capture (#match? @capture "^[A-Z]"))`, nil},
		{"#not-match?: too few arguments", `((expression) @capture (#not-match? "this"))`, ErrPredicateArgsWrongCount},
		{"#not-match?: too many arguments", `((expression) @capture (#not-match? a b "this"))`, ErrPredicateArgsWrongCount},
		{"#not-match?: need a capture as first argument", `((expression) @capture (#not-match? "a" "this"))`, ErrPredicateWrongType},
		{"#not-match?: need a string as second argument", `((expression) @capture (#not-match? @capture @capture))`, ErrPredicateWrongType},
		{"#not-match?: success test", `((expression) @capture (#not-match? @capture "^[A-Z]"))`, nil},
		{"#eq?: too few arguments", `((expression) @capture (#eq? "this"))`, ErrPredicateArgsWrongCount},
		{"#eq?: too many arguments", `((expression) @capture (#eq? a b "this"))`, ErrPredicateArgsWrongCount},
		{"#eq?: need a capture as first argument", `((expression) @capture (#eq? "a" "this"))`, ErrPredicateWrongType},
		{"#eq?: success test", `((expression) @capture (#eq? @capture "this"))`, nil},
		{"#eq?: success double predicate test", `((expression) @capture (#eq? @capture @capture) (#eq? @capture "this"))`, nil},
		{"#eq?: success test", `((expression) @capture (#eq? @capture @capture))`, nil},
		{"#not-eq?: too few arguments", `((expression) @capture (#not-eq? "this"))`, ErrPredicateArgsWrongCount},
		{"#not-eq?: too many arguments", `((expression) @capture (#not-eq? a b "this"))`, ErrPredicateArgsWrongCount},
		{"#not-eq?: need a capture as first argument", `((expression) @capture (#not-eq? "a" "this"))`, ErrPredicateWrongType},
		{"#not-eq?: success test", `((expression) @capture (#not-eq? @capture "this"))`, nil},
		{"#not-eq?: success test", `((expression) @capture (#not-eq? @capture @capture))`, nil},
		{"#is?: too few arguments", `((expression) @capture (#is?))`, ErrPredicateArgsWrongCount},
		{"#is?: too many arguments", `((expression) @capture (#is? a b "this"))`, ErrPredicateArgsWrongCount},
		{"#is?: need a string as first argument", `((expression) @capture (#is? @capture "this"))`, ErrPredicateWrongType},
		{"#is?: need a string as second argument", `((expression) @capture (#is? "this" @capture))`, ErrPredicateWrongType},
		{"#is?: success test", `((expression) @capture (#is? "foo" "bar"))`, nil},
		{"#is-not?: too few arguments", `((expression) @capture (#is-not?))`, ErrPredicateArgsWrongCount},
		{"#is-not?: too many arguments", `((expression) @capture (#is-not? a b "this"))`, ErrPredicateArgsWrongCount},
		{"#is-not?: need a string as first argument", `((expression) @capture (#is-not? @capture "this"))`, ErrPredicateWrongType},
		{"#is-not?: need a string as second argument", `((expression) @capture (#is-not? "this" @capture))`, ErrPredicateWrongType},
		{"#is-not?: success test", `((expression) @capture (#is-not? "foo" "bar"))`, nil},
		{"#set!: too few arguments", `((expression) @capture (#set!))`, ErrPredicateArgsWrongCount},
		{"#set!: too many arguments", `((expression) @capture (#set! a b "this"))`, ErrPredicateArgsWrongCount},
		{"#set!: need a string as first argument", `((expression) @capture (#set! @capture "this"))`, ErrPredicateWrongType},
		{"#set!: need a string as second argument", `((expression) @capture (#set! "this" @capture))`, ErrPredicateWrongType},
		{"#set!: success test", `((expression) @capture (#set! "foo" "bar"))`, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			t.Parallel()

			q, err := NewQuery([]byte(tc.pattern), getTestGrammar())
			success := strings.Contains(tc.msg, "success")

			if (err != nil) && success {
				t.Fatal(tc.msg)
			}

			if !errors.Is(err, tc.exp) {
				t.Fatalf("Expected %v, got %v", tc.exp, err)
			}

			// Also add tests for actual error message.

			if (q == nil) && success {
				t.Fatal(tc.msg)
			}
		})
	}
}

func TestFilterPredicates(t *testing.T) {
	t.Parallel()

	sumLR := `((sum left: (expression (number) @left) right: (expression (number) @right))`
	testCases := []struct {
		input, query        string
		expBefore, expAfter int
	}{
		{`// foo`, `((comment) @capture (#match? @capture "^// [a-z]+$"))`, 1, 1},
		{`// foo123`, `((comment) @capture (#match? @capture "^// [a-z]+$"))`, 1, 0},
		{`// foo`, `((comment) @capture (#not-match? @capture "^// [a-z]+$"))`, 1, 0},
		{`// foo123`, `((comment) @capture (#not-match? @capture "^// [a-z]+$"))`, 1, 1},
		{`// foo`, `((comment) @capture (#eq? @capture "// foo"))`, 1, 1},
		{`// foo`, `((comment) @capture (#eq? @capture "// bar"))`, 1, 0},
		{`// foo`, `((comment) @capture (#eq? @capture "// foo") (#eq? @capture "// bar"))`, 1, 0},
		{`1234 + 1234`, sumLR + ` (#eq? @left @right))`, 2, 2},
		{`1234 + 4321`, sumLR + ` (#eq? @left @right))`, 2, 0},
		{`// foo`, `((comment) @capture (#not-eq? @capture "// foo"))`, 1, 0},
		{`// foo`, `((comment) @capture (#not-eq? @capture "// bar"))`, 1, 1},
		{`1234 + 1234`, sumLR + ` (#not-eq? @left @right))`, 2, 0},
		{`1234 + 4321`, sumLR + ` (#not-eq? @left @right))`, 2, 2},
		{`1234 + 4321`, sumLR + ` (#eq? @left 1234))`, 2, 2},
		{`1234 + 4321`, sumLR + ` (#eq? @left 1234) (#not-eq? @left @right))`, 2, 2},
		{`1234 + 4321`, sumLR + ` (#eq? @left 1234) (#eq? @left 4321))`, 2, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			p := NewParser()
			p.SetLanguage(getTestGrammar())

			tree, err := p.ParseString(context.TODO(), nil, []byte(tc.input))
			if err != nil {
				t.Fatal("Expected no error, got", err)
			}

			root := tree.RootNode()

			q, err := NewQuery([]byte(tc.query), getTestGrammar())
			if err != nil {
				t.Fatal("Expected no error, got", err)
			}

			qc := NewQueryCursor()
			qc.Exec(q, root)

			before, ok := qc.NextMatch()
			if !ok {
				t.Fatal("Expected a match, got none for", tc.query)
			}

			if x := len(before.Captures); x != tc.expBefore {
				t.Fatalf("Expected %d captures before filtering, got %d", tc.expBefore, x)
			}

			after := qc.FilterPredicates(before, []byte(tc.input))
			if x := len(after.Captures); x != tc.expAfter {
				t.Fatalf("Expected %d captures after filtering, got %d", tc.expAfter, x)
			}
		})
	}
}
