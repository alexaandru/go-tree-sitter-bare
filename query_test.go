package sitter

import (
	"context"
	"strings"
	"testing"
)

func TestQueryWithPredicates(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		msg,
		pattern string
	}{
		{"#match?: too few arguments", `((expression) @capture (#match? "this"))`},
		{"#match?: too many arguments", `((expression) @capture (#match? a b "this"))`},
		{"#match?: need a capture as first argument", `((expression) @capture (#match? "a" "this"))`},
		{"#match?: need a string as second argument", `((expression) @capture (#match? @capture @capture))`},
		{"#match?: success test", `((expression) @capture (#match? @capture "^[A-Z]"))`},
		{"#not-match?: too few arguments", `((expression) @capture (#not-match? "this"))`},
		{"#not-match?: too many arguments", `((expression) @capture (#not-match? a b "this"))`},
		{"#not-match?: need a capture as first argument", `((expression) @capture (#not-match? "a" "this"))`},
		{"#not-match?: need a string as second argument", `((expression) @capture (#not-match? @capture @capture))`},
		{"#not-match?: success test", `((expression) @capture (#not-match? @capture "^[A-Z]"))`},
		{"#eq?: too few arguments", `((expression) @capture (#eq? "this"))`},
		{"#eq?: too many arguments", `((expression) @capture (#eq? a b "this"))`},
		{"#eq?: need a capture as first argument", `((expression) @capture (#eq? "a" "this"))`},
		{"#eq?: success test", `((expression) @capture (#eq? @capture "this"))`},
		{"#eq?: success double predicate test", `((expression) @capture (#eq? @capture @capture) (#eq? @capture "this"))`},
		{"#eq?: success test", `((expression) @capture (#eq? @capture @capture))`},
		{"#not-eq?: too few arguments", `((expression) @capture (#not-eq? "this"))`},
		{"#not-eq?: too many arguments", `((expression) @capture (#not-eq? a b "this"))`},
		{"#not-eq?: need a capture as first argument", `((expression) @capture (#not-eq? "a" "this"))`},
		{"#not-eq?: success test", `((expression) @capture (#not-eq? @capture "this"))`},
		{"#not-eq?: success test", `((expression) @capture (#not-eq? @capture @capture))`},
		{"#is?: too few arguments", `((expression) @capture (#is?))`},
		{"#is?: too many arguments", `((expression) @capture (#is? a b "this"))`},
		{"#is?: need a string as first argument", `((expression) @capture (#is? @capture "this"))`},
		{"#is?: need a string as second argument", `((expression) @capture (#is? "this" @capture))`},
		{"#is?: success test", `((expression) @capture (#is? "foo" "bar"))`},
		{"#is-not?: too few arguments", `((expression) @capture (#is-not?))`},
		{"#is-not?: too many arguments", `((expression) @capture (#is-not? a b "this"))`},
		{"#is-not?: need a string as first argument", `((expression) @capture (#is-not? @capture "this"))`},
		{"#is-not?: need a string as second argument", `((expression) @capture (#is-not? "this" @capture))`},
		{"#is-not?: success test", `((expression) @capture (#is-not? "foo" "bar"))`},
		{"#set!: too few arguments", `((expression) @capture (#set!))`},
		{"#set!: too many arguments", `((expression) @capture (#set! a b "this"))`},
		{"#set!: need a string as first argument", `((expression) @capture (#set! @capture "this"))`},
		{"#set!: need a string as second argument", `((expression) @capture (#set! "this" @capture))`},
		{"#set!: success test", `((expression) @capture (#set! "foo" "bar"))`},
	}

	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			t.Parallel()

			q, err := NewQuery([]byte(tc.pattern), getTestGrammar())
			success := strings.Contains(tc.msg, "success")

			if (err != nil) && success {
				t.Fatal(tc.msg)
			}

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
		input,
		query string
		expBefore,
		expAfter int
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
