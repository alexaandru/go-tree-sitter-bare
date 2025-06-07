package sitter

import "testing"

func TestLanguageCopy(t *testing.T) {
	t.Parallel()

	gr2 := gr.Copy()

	if gr.ptr != gr2.ptr {
		t.Fatal("The two grammars differ")
	}
}

func TestLanguageDelete(t *testing.T) {
	t.Parallel()

	gr2 := gr.Copy()

	gr2.Delete()

	// Not sure what else I could test.
	// How could I possibly test that C freed the memory?
	if gr2.ptr != nil {
		t.Fatal("Expected gr2 to be deleted")
	}
}

func TestLanguage(t *testing.T) {
	t.Parallel()

	exp := uint32(9)
	if x := gr.SymbolCount(); x != exp {
		t.Fatalf("Expected symbol count to be %d, got %d", exp, x)
	}

	expStr := "Regular"
	if x := SymbolTypeRegular.String(); x != expStr {
		t.Fatalf("Expected regular symbol type to be %q, got %q", expStr, x)
	}

	testCases := []struct {
		n       Symbol
		expName string
		expType SymbolType
		isNamed bool
	}{
		{0, "end", SymbolTypeAuxiliary, true},
		{1, "(", SymbolTypeAnonymous, false},
		{2, ")", SymbolTypeAnonymous, false},
		{3, "+", SymbolTypeAnonymous, false},
		{4, "number", SymbolTypeRegular, true},
		{5, "comment", SymbolTypeRegular, true},
		{6, "variable", SymbolTypeRegular, true},
		{7, "expression", SymbolTypeRegular, true},
		{8, "sum", SymbolTypeRegular, true},
		{9, "", SymbolTypeAuxiliary, false},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			symName := gr.SymbolName(tc.n)
			if symName != tc.expName {
				t.Fatalf("Expected symbol name %q got %q for %d", tc.expName, symName, tc.n)
			}

			symID := gr.SymbolID(symName, tc.isNamed)
			if symName != "" && symID != tc.n {
				t.Fatalf("Expected symbol ID %d got %d for %q", tc.n, symID, symName)
			}

			symType := gr.SymbolType(tc.n)
			if symType != tc.expType {
				t.Fatalf("Expected symbol type %d got %d for %d", tc.expType, symType, tc.n)
			}
		})
	}
}

func TestLanguageSymbolCount(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly via TestLanguage()")
}

func TestLanguageStateCount(t *testing.T) {
	t.Parallel()

	exp := uint32(9)
	if act := gr.StateCount(); act != exp {
		t.Fatalf("Expected state count to be %d, got %d", exp, act)
	}
}

func TestLanguageSymbolName(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly via TestLanguage()")
}

func TestLanguageSymbolID(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly via TestLanguage()")
}

func TestLanguageFieldCount(t *testing.T) {
	t.Parallel()

	exp := uint32(2)
	if act := gr.FieldCount(); act != exp {
		t.Fatalf("Expected field count to be %d, got %d", exp, act)
	}
}

func TestLanguageFieldName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		n   int
		exp string
	}{
		{0, ""},
		{1, "left"},
		{2, "right"},
	}

	for _, tc := range testCases {
		t.Run(tc.exp, func(t *testing.T) {
			t.Parallel()

			if act := gr.FieldName(tc.n); act != tc.exp {
				t.Fatalf("Expected %q, got %q for %d", tc.exp, act, tc.n)
			}
		})
	}
}

func TestLanguageFieldID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		exp FieldID
		n   string
	}{
		{0, ""},
		{1, "left"},
		{2, "right"},
	}

	for _, tc := range testCases {
		t.Run(tc.n, func(t *testing.T) {
			t.Parallel()

			if act := gr.FieldID(tc.n); act != tc.exp {
				t.Fatalf("Expected %d, got %q for %q", tc.exp, act, tc.n)
			}
		})
	}
}

func TestLanguageSymbolType(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly via TestLanguage()")
}

func TestLanguageABIVersion(t *testing.T) {
	t.Parallel()

	exp := TREE_SITTER_LANGUAGE_VERSION
	if act := gr.ABIVersion(); act != exp {
		t.Fatalf("Expected %d, got %d", exp, act)
	}
}

func TestLanguageMetadata(t *testing.T) {
	t.Parallel()

	exp := LanguageMetadata{15, 1, 2}
	if act := gr.Metadata(); act != exp {
		t.Fatalf("Expected %v, got %v", exp, act)
	}
}

func TestLanguageName(t *testing.T) {
	t.Parallel()

	exp := "test_grammar"
	if act := gr.Name(); act != exp {
		t.Fatalf("Expected %q, got %q", exp, act)
	}
}

func TestLanguageNextState(t *testing.T) {
	t.Parallel()
	t.Skip("won't test: not really the Go code's job to assert the parser parses correctly")
}
