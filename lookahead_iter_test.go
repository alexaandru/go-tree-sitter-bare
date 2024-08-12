package sitter

import (
	"context"
	"testing"
)

func TestNewLookaheadIterator(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly via TestLookaheadIteratorNext()")
}

func TestLookaheadIteratorDelete(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly via TestLookaheadIteratorNext()")
}

func TestLookaheadIteratorResetState(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestLookaheadIteratorReset(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestLookaheadIteratorLanguage(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly via TestLookaheadIteratorNext()")
}

func TestLookaheadIteratorNext(t *testing.T) {
	t.Parallel()

	source := "1 + "

	input := []byte(source)

	root, err := Parse(context.Background(), input, gr)
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	t.Log(root)

	c := NewTreeCursor(root)
	ok1 := c.GoToFirstChild()
	ok2 := c.GoToFirstChild()
	ok3 := c.GoToNextSibling()

	if !ok1 || !ok2 || !ok3 {
		t.Fatal("Node navigation failed")
	}

	iter := NewLookaheadIterator(gr, c.CurrentNode().ParseState())
	if iter == nil {
		t.Fatal("Could not get iter")
	}

	defer iter.Delete()

	gr2 := iter.Language()

	if gr2.ptr != gr.ptr {
		t.Fatal("The language differs")
	}

	testCases := []struct {
		expSym  uint16
		expName string
	}{
		{maxUint16, "ERROR"},
		{5, "comment"},
		{0, "end"},
		{2, ")"},
		{3, "+"},
	}

	for i, tc := range testCases {
		if actSym := iter.CurrentSymbol(); uint16(actSym) != tc.expSym {
			t.Fatalf("Expected symbol %d, got %d", tc.expSym, actSym)
		}

		if actName := iter.CurrentSymbolName(); actName != tc.expName {
			t.Fatalf("Expected symbol name %q, got %q", tc.expName, actName)
		}

		if i == len(testCases)-1 {
			return
		}

		if !iter.Next() {
			t.Fatalf("Iterator could not advance from %v", tc)
		}
	}
}

func TestLookaheadIteratorCurrentSymbol(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly via TestLookaheadIteratorNext()")
}

func TestLookaheadIteratorCurrentSymbolName(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly via TestLookaheadIteratorNext()")
}
