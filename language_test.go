package sitter

import "testing"

func TestLanguageCopy(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestLanguageDelete(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestLanguage(t *testing.T) {
	t.Parallel()

	js := getTestGrammar()

	exp := uint32(9)
	if x := js.SymbolCount(); x != exp {
		t.Fatalf("Expected symbol count to be %d, got %d", exp, x)
	}

	expStr := "Regular"
	if x := SymbolTypeRegular.String(); x != expStr {
		t.Fatalf("Expected regular symbol type to be %q, got %q", expStr, x)
	}

	testCases := []struct {
		n       uint16
		expName string
		expType SymbolType
	}{
		{0, "end", SymbolTypeAuxiliary},
		{1, "(", SymbolTypeAnonymous},
		{2, ")", SymbolTypeAnonymous},
		{3, "+", SymbolTypeAnonymous},
		{4, "number", SymbolTypeRegular},
		{5, "comment", SymbolTypeRegular},
		{6, "variable", SymbolTypeRegular},
		{7, "expression", SymbolTypeRegular},
		{8, "sum", SymbolTypeRegular},
		{9, "", SymbolTypeAuxiliary},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			symName := js.SymbolName(Symbol(tc.n))
			if symName != tc.expName {
				t.Fatalf("Expected symbol name %q got %q for %d", tc.expName, symName, tc.n)
			}

			symType := js.SymbolType(Symbol(tc.n))
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
	t.Skip("TODO")
}

func TestLanguageSymbolName(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly via TestLanguage()")
}

func TestLanguageSymbolID(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestLanguageFieldCount(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestLanguageFieldName(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestLanguageFieldID(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestLanguageSymbolType(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly via TestLanguage()")
}

func TestLanguageVersion(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}

func TestLanguageNextState(t *testing.T) {
	t.Parallel()
	t.Skip("TODO")
}
