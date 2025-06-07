package sitter

import (
	"reflect"
	"testing"
)

type (
	seqTestCase[T any] struct {
		nav string
		exp T
	}

	seqTestCases[T any] struct {
		op  string
		seq []seqTestCase[T]
	}
)

func TestSymbolString(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeType(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeSymbol(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeLanguage(t *testing.T) {
	t.Parallel()

	input := []byte(`1 + 2`)

	root, err := Parse(t.Context(), input, gr)
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	exp := uintptr(gr.ptr)

	act := uintptr(root.Language().ptr)
	if act != exp {
		t.Fatalf("The languages differ")
	}

	c := NewTreeCursor(root)

	c.GoToFirstChild()

	act = uintptr(c.CurrentNode().Language().ptr)
	if act != exp {
		t.Fatalf("The languages differ")
	}
}

func TestNodeGrammarType(t *testing.T) {
	t.Parallel()
	testParserSequence(t, "1 + 2", seqTestCases[string]{
		op: "GrammarType", seq: []seqTestCase[string]{
			{"", "expression"},
			{"FirstChild", "sum"},
			{"FirstChild", "expression"},
			{"FirstChild", "number"},
		},
	})
}

func TestNodeGrammarSymbol(t *testing.T) {
	t.Parallel()
	testParserSequence(t, "1 + 2", seqTestCases[Symbol]{
		op: "GrammarSymbol", seq: []seqTestCase[Symbol]{
			{"", 7},
			{"FirstChild", 8},
			{"FirstChild", 7},
			{"FirstChild", 4},
		},
	})
}

func TestNodeStartByte(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeStartPoint(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeEndByte(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeEndPoint(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeString(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeIsNull(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeIsNamed(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeIsMissing(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeIsExtra(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeHasChanges(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeHasError(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeIsError(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeParseState(t *testing.T) {
	t.Parallel()
	testParserSequence(t, "1 + 2", seqTestCases[StateID]{
		op: "ParseState", seq: []seqTestCase[StateID]{
			{"", 0},
			{"FirstChild", 1},
			{"FirstChild", 1},
			{"FirstChild", 1},
		},
	})
}

func TestNodeNextParseState(t *testing.T) {
	t.Parallel()
	testParserSequence(t, "1 + 2", seqTestCases[StateID]{
		op: "NextParseState", seq: []seqTestCase[StateID]{
			{"", 0},
			{"FirstChild", 4},
			{"FirstChild", 7},
			{"FirstChild", 4},
		},
	})
}

func TestNodeParent(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeChildWithDescendant(t *testing.T) {
	t.Parallel()

	input := []byte(`1 + 2`)

	root, err := Parse(t.Context(), input, gr)
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	c := NewTreeCursor(root)
	if c.CurrentNode() != root {
		t.Fatal("Expected current node to be root")
	}

	c.GoToFirstChild()

	n := c.CurrentNode()

	exp := "(sum left: (expression (number)) right: (expression (number)))"
	if act := n.String(); act != exp {
		t.Fatalf("Expected %q, got %q", exp, act)
	}

	c.GoToFirstChild()
	c.GoToNextSibling()
	c.GoToNextSibling()
	c.GoToFirstChild()

	d := c.CurrentNode()

	exp = "(number)"
	if act := d.String(); act != exp {
		t.Fatalf("Expected %q, got %q", exp, act)
	}

	c.GoToParent()

	p := c.CurrentNode()

	exp = "(expression (number))"
	if act := p.String(); act != exp {
		t.Fatalf("Expected %q, got %q", exp, act)
	}

	if act := n.ChildWithDescendant(d); act != p {
		t.Fatalf("Expected %v, got %v", p, act)
	}
}

func TestNodeChild(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeFieldNameForChild(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeFieldNameForNamedChild(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestNodeChildCount(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeNamedChild(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeNamedChildCount(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeChildByFieldName(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeChildByFieldID(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestNodeNextSibling(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodePrevSibling(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeNextNamedSibling(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodePrevNamedSibling(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeFirstChildForByte(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestNodeFirstNamedChildForByte(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestNodeDescendantCount(t *testing.T) {
	t.Parallel()
	testParserSequence(t, "1 + 2", seqTestCases[uint32]{
		op: "DescendantCount", seq: []seqTestCase[uint32]{
			{"", 7},
			{"FirstChild", 6},
			{"FirstChild", 2},
			{"FirstChild", 1},
		},
	})
}

func TestNodeDescendantForByteRange(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestNodeDescendantForPointRange(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestNodeNamedDescendantForByteRange(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestNodeNamedDescendantForPointRange(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeEdit(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestNodeEqual(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestNodeRange(t *testing.T) {
	t.Parallel()
	testParserSequence(t, "1 + 2", seqTestCases[Range]{
		op: "Range", seq: []seqTestCase[Range]{
			{"", Range{EndPoint: Point{Column: 5}, EndByte: 5}},
			{"FirstChild", Range{EndPoint: Point{Column: 5}, EndByte: 5}},
			{"FirstChild", Range{EndPoint: Point{Column: 1}, EndByte: 1}},
			{"FirstChild", Range{EndPoint: Point{Column: 1}, EndByte: 1}},
		},
	})
}

func TestNodeContent(t *testing.T) {
	t.Parallel()
	testParserSequence(t, "1 + 2", seqTestCases[string]{
		op: "Content", seq: []seqTestCase[string]{
			{"", "1 + 2"},
			{"FirstChild", "1 + 2"},
			{"FirstChild", "1"},
			{"FirstChild", "1"},
		},
	})
}

func testParserSequence[T any](t *testing.T, source string, testCases seqTestCases[T]) {
	t.Helper()

	input := []byte(source)

	root, err := Parse(t.Context(), input, gr)
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	t.Log(root)

	c := NewTreeCursor(root)

	for i, tc := range testCases.seq {
		t.Run("", func(t *testing.T) {
			ok := false

			switch tc.nav {
			case "": // noop
				ok = true
			case "FirstChild":
				ok = c.GoToFirstChild()
			default:
				t.Fatalf("Unknown operation %q", tc.nav)
			}

			if !ok {
				t.Fatalf("Failed to navigate to node #%d", i+1)
			}

			nn := c.CurrentNode()

			var act any

			switch testCases.op {
			case "GrammarSymbol":
				act = any(nn.GrammarSymbol())
			case "GrammarType":
				act = any(nn.GrammarType())
			case "ParseState":
				act = any(nn.ParseState())
			case "NextParseState":
				act = any(nn.NextParseState())
			case "DescendantCount":
				act = any(nn.DescendantCount())
			case "Range":
				act = any(nn.Range())
			case "Content":
				act = any(nn.Content(input))
			default:
				t.Fatalf("Unknown method %q", testCases.op)
			}

			if !eq(act, tc.exp) {
				t.Fatalf("Expected %v, got %v", tc.exp, act)
			}
		})
	}
}

func eq(a, b any) bool {
	return reflect.DeepEqual(a, b)
}
