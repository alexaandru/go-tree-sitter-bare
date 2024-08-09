package sitter

import (
	"context"
	"testing"
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
	t.Skip("TODO")
}

func TestNodeGrammarType(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestNodeGrammarSymbol(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
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
	t.Skip("TODO")
}

func TestNodeNextParseState(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestNodeParent(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestNodeChildContainingDescendant(t *testing.T) {
	t.Parallel()

	input := []byte(`1 + 2`)

	root, err := Parse(context.Background(), input, getTestGrammar())
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

	if act := n.ChildContainingDescendant(d); act != p {
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
	t.Skip("TODO")
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

func TestNodeID(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestNodeRange(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestNodeContent(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}
