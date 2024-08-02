package sitter

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	exprSumLR = "(expression (sum left: (expression (number)) right: (expression (number))))"
	// NOTE: We have a lot of parsers created in parallal so the increase in
	// memory usage seems normal.
	leakLimit = 25 * 1024 * 1024
)

func TestRootNode(t *testing.T) {
	t.Parallel()

	n, err := Parse(context.Background(), []byte("1 + 2"), getTestGrammar())
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	testStartEnd(t, n, 0, 5, 0, 0, 0, 5)

	if x := n.String(); x != exprSumLR {
		t.Fatalf("Expected tree to be %q, got %q", exprSumLR, x)
	}

	expType := "expression"
	if x := n.Type(); x != expType {
		t.Fatalf("Expected type to be %q, got %q", expType, x)
	}

	expSymbol := Symbol(7)
	if x := n.Symbol(); x != expSymbol {
		t.Fatalf("Expected symbol to be %v, got %v", expSymbol, x)
	}

	if !n.IsNamed() {
		t.Fatal("Expected tree to be named")
	}

	for _, fn := range []func() bool{n.IsNull, n.IsMissing, n.IsExtra, n.IsError, n.HasChanges, n.HasError} {
		name := nameOf(fn)

		t.Run(name, func(t *testing.T) {
			if fn() {
				t.Fatalf("Expected n.%s() == false, got true", name)
			}
		})
	}

	if x := n.ChildCount(); x != 1 {
		t.Fatalf("Expected n.ChildCount() == 1, got %d", x)
	}

	if x := n.NamedChildCount(); x != 1 {
		t.Fatalf("Expected n.NamedChildCount() == 1, got %d", x)
	}

	for _, fn := range []func() *Node{
		n.Parent, n.NextSibling, n.NextNamedSibling, n.PrevSibling, n.PrevNamedSibling,
		func() *Node { return n.ChildByFieldName("unknown") },
	} {
		name := nameOf(fn)

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if x := fn(); x != nil {
				t.Fatalf("Expected n.%s() == nil, got %v", name, x)
			}
		})
	}

	if n.Child(0) == nil {
		t.Fatalf("Expected n.Child(0) to not be nil")
	}

	if n.NamedChild(0) == nil {
		t.Fatalf("Expected n.NamedChild(0) to not be nil")
	}

	if n.NamedChild(0).ChildByFieldName("left") == nil {
		t.Fatalf(`Expected n.NamedChild(0).ChildByFieldName("left") to not be nil`)
	}
}

func TestTree(t *testing.T) {
	t.Parallel()

	parser := NewParser()
	parser.Debug()
	parser.SetLanguage(getTestGrammar())

	tree, err := parser.ParseString(context.Background(), nil, []byte("1 + 2"))
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	n := tree.RootNode()
	sum := n.Child(0)

	testStartEnd(t, n, 0, 5, 0, 0, 0, 5)

	if x := n.String(); x != exprSumLR {
		t.Fatalf("Expected tree to be %q, got %q", exprSumLR, x)
	}

	expType := "expression"
	if x := n.Type(); x != expType {
		t.Fatalf("Expected type to be %q, got %q", expType, x)
	}

	expType = "(expression (number))"
	if x := sum.Child(0).String(); x != expType {
		t.Fatalf("Expected type to be %q, got %q", expType, x)
	}

	if x := sum.Child(2).String(); x != expType {
		t.Fatalf("Expected type to be %q, got %q", expType, x)
	}

	expType = "left"
	if x := sum.FieldNameForChild(0); x != expType {
		t.Fatalf("Expected type to be %q, got %q", expType, x)
	}

	expType = "right"
	if x := sum.FieldNameForChild(2); x != expType {
		t.Fatalf("Expected type to be %q, got %q", expType, x)
	}

	expType = ""
	if x := sum.FieldNameForChild(100); x != expType {
		t.Fatalf("Expected type to be %q, got %q", expType, x)
	}

	// change 2 -> (3 + 3)
	newText := []byte("1 + (3 + 3)")

	tree.Edit(InputEdit{
		StartIndex:  4,
		OldEndIndex: 5,
		NewEndIndex: 11,
		StartPoint: Point{
			Row:    0,
			Column: 4,
		},
		OldEndPoint: Point{
			Row:    0,
			Column: 5,
		},
		NewEndPoint: Point{
			Row:    0,
			Column: 11,
		},
	})

	rngExp := []Range{{EndPoint: Point{Row: maxUint32, Column: maxUint32}, EndByte: maxUint32}}
	// Testing that it doesn't crash, as it involves a `C.free()`.
	for range 10_000 {
		if act := tree.IncludedRanges(); !slices.Equal(rngExp, act) {
			t.Fatalf("Expected\n\n%#v, got\n\n%#v\n", rngExp, act)
		}
	}

	// check that it changed tree
	if !n.HasChanges() {
		t.Fatal("Expected tree to have changes")
	}

	if !n.Child(0).HasChanges() {
		t.Fatal("Expected 1st child to have changes")
	}

	if n.Child(0).Child(0).HasChanges() { // left side of the sum didn't change
		t.Fatal("Expected no changes for 1st grandchild")
	}

	if !n.Child(0).Child(2).HasChanges() {
		t.Fatal("Expected changes for 3st grandchild")
	}

	tree2, err := parser.ParseString(context.Background(), tree, newText)
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	n = tree2.RootNode()

	expString := "(expression (sum left: (expression (number)) right: (expression " + exprSumLR + ")))"
	if x := n.String(); x != expString {
		t.Fatalf("Expected %q got %q", expString, x)
	}

	descendantNode := n.NamedDescendantForPointRange(Point{Row: 0, Column: 5}, Point{Row: 0, Column: 11})
	if descendantNode == nil {
		t.Fatalf("Expected descendent node to not be nil")
	}

	expContent := "(3 + 3)"
	if x := descendantNode.Content(newText); x != expContent {
		t.Fatalf("Expected descendent content to be %q got %q", expContent, x)
	}
}

func TestErrorNodes(t *testing.T) {
	t.Parallel()

	parser := NewParser()

	parser.Debug()
	parser.SetLanguage(getTestGrammar())

	tree, err := parser.ParseString(context.Background(), nil, []byte("1 + a"))
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	n := tree.RootNode()

	if !n.HasError() {
		t.Fatal("Expected error")
	}

	exp := "(expression (number) (ERROR (UNEXPECTED '\\0')))"
	if act := n.String(); act != exp {
		t.Fatalf("Expected %q, got %q", exp, act)
	}

	number := n.Child(0)

	if number.IsError() || number.HasError() {
		t.Fatal("Expected no error")
	}

	errorNode := n.Child(1)

	if !errorNode.HasError() {
		t.Fatal("Expected error")
	}

	if !errorNode.IsError() {
		t.Fatal("Expected error")
	}

	tree, err = parser.ParseString(context.Background(), nil, []byte("1 +"))
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	n = tree.RootNode()

	if !n.HasError() {
		t.Fatal("Expected error")
	}

	exp = "(expression (sum left: (expression (number)) right: (expression (MISSING number))))"
	if act := n.String(); act != exp {
		t.Fatalf("Expected %q, got %q", exp, act)
	}

	sum := n.Child(0)

	if !sum.HasError() {
		t.Fatal("Expected error")
	}

	left := sum.Child(0)

	if left.HasError() {
		t.Fatal("Expected no error")
	}

	right := sum.Child(2)

	if !right.HasError() {
		t.Fatal("Expected error")
	}

	if right.IsError() {
		t.Fatal("Expected no error")
	}

	missing := right.Child(0)

	if !missing.HasError() {
		t.Fatal("Expected error")
	}

	if missing.IsError() {
		t.Fatal("Expected no error")
	}

	if !missing.IsMissing() {
		t.Fatal("Expected missing")
	}
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
		expType Symbol
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

func TestGC(t *testing.T) {
	t.Parallel()

	parser := NewParser()
	parser.SetLanguage(getTestGrammar())

	tree, err := parser.ParseString(context.Background(), nil, []byte("1 + 2"))
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	n := tree.RootNode()
	if !isNamedWithGC(n) {
		t.Fatal("Expected n.IsNamed() with GC to be true")
	}
}

func TestSetOperationLimit(t *testing.T) {
	t.Parallel()

	parser := NewParser()
	if x := parser.TimeoutMicros(); x != 0 {
		t.Fatalf("Expected parser.OperationLimit() == 0, got %d", x)
	}

	parser.SetTimeoutMicros(10)

	if x := parser.TimeoutMicros(); x != 10 {
		t.Fatalf("Expected parser.OperationLimit() == 10, got %d", x)
	}
}

func TestOperationLimitParsing(t *testing.T) {
	t.Parallel()

	parser := NewParser()
	parser.SetTimeoutMicros(10)
	parser.SetLanguage(getTestGrammar())

	items := []string{}

	for i := range 100 {
		items = append(items, strconv.Itoa(i))
	}

	code := strings.Join(items, " + ")

	tree, err := parser.ParseString(context.Background(), nil, []byte(code))
	if !errors.Is(err, ErrOperationLimit) {
		t.Fatalf("Expected error to be %v, got %v", ErrOperationLimit, err)
	}

	if tree != nil {
		t.Fatal("Expected tree to be nil, got", tree)
	}
}

func TestContextCancellationParsing(t *testing.T) {
	t.Parallel()

	parser := NewParser()
	parser.SetLanguage(getTestGrammar())

	items := []string{}

	// the content needs to be big so that we have enough time to cancel
	for i := range 10_000 {
		items = append(items, strconv.Itoa(i))
	}

	code := strings.Join(items, " + ")
	started, done := make(chan bool), make(chan bool)

	var (
		tree *Tree
		err  error
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer close(started)
		defer close(done)

		start := time.Now()
		started <- true

		tree, err = parser.ParseString(ctx, nil, []byte(code))

		t.Logf("parsing complete after %s, error: %+v\n", time.Since(start), err)

		done <- true
	}()

	<-started
	cancel()
	<-done

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected error to be %v, got %v", context.Canceled, err)
	}

	if tree != nil {
		t.Fatal("Expected tree to be nil, got", tree)
	}

	// make sure we can re-use parse after cancellation
	ctx = context.Background()
	tree, err = parser.ParseString(ctx, nil, []byte("1 + 1"))

	if tree == nil {
		t.Fatal("Expected tree to not be nil")
	}

	if err != nil {
		t.Fatal("Expected error to be nil, got", err)
	}
}

func TestIncludedRanges(t *testing.T) {
	t.Parallel()

	// sum code with sum code in a comment
	code := "1 + 2\n//3 + 5"

	parser := NewParser()
	parser.SetLanguage(getTestGrammar())

	mainTree, err := parser.ParseString(context.Background(), nil, []byte(code))
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	expString := "(expression (sum left: (expression (number)) right: (expression (number))) (comment))"
	if x := mainTree.RootNode().String(); x != expString {
		t.Fatalf("Expected root node to be %q, got %q", expString, x)
	}

	commentNode := mainTree.RootNode().NamedChild(1)

	expType := "comment"
	if x := commentNode.Type(); x != expType {
		t.Fatalf("Expected comment node's type to be %q, got %q", expType, x)
	}

	commentRange := Range{
		StartPoint: Point{
			Row:    commentNode.StartPoint().Row,
			Column: commentNode.StartPoint().Column + 2,
		},
		EndPoint:  commentNode.EndPoint(),
		StartByte: commentNode.StartByte() + 2,
		EndByte:   commentNode.EndByte(),
	}

	parser.SetIncludedRanges([]Range{commentRange})

	commentTree, err := parser.ParseString(context.Background(), nil, []byte(code))
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	if x := commentTree.RootNode().String(); x != exprSumLR {
		t.Fatalf("Expected root node to be %q, got %q", exprSumLR, x)
	}
}

func TestSameNode(t *testing.T) {
	t.Parallel()

	parser := NewParser()
	parser.SetLanguage(getTestGrammar())

	tree, err := parser.ParseString(context.Background(), nil, []byte("1 + 2"))
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	n1, n2 := tree.RootNode(), tree.RootNode()
	if n1 != n2 {
		t.Fatal("Expected n1 and n2 to be equal", n1, n2)
	}

	n1 = tree.RootNode().NamedChild(0)
	n2 = tree.RootNode().NamedChild(0)

	if n1 != n2 {
		t.Fatal("Expected n1 and n2 to be equal", n1, n2)
	}
}

func TestQuery(t *testing.T) {
	t.Parallel()

	js := "1 + 2"

	// test single capture
	testCaptures(t, js, "(sum left: (expression) @left)", []string{
		"1",
	})

	// test multiple captures
	testCaptures(t, js, "(sum left: _* @left right: _* @right)", []string{
		"1",
		"2",
	})

	// test match only
	parser := NewParser()
	parser.SetLanguage(getTestGrammar())

	tree, err := parser.ParseString(context.Background(), nil, []byte(js))
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	root := tree.RootNode()

	q, err := NewQuery([]byte("(sum) (number)"), getTestGrammar())
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	qc := NewQueryCursor()
	qc.Exec(q, root)

	var matched int

	for {
		_, ok := qc.NextMatch()
		if !ok {
			break
		}

		matched++
	}

	exp := 3
	if matched != exp {
		t.Fatalf("Expected %d matches, got %d", exp, matched)
	}
}

func TestQueryError(t *testing.T) {
	t.Parallel()

	q, err := NewQuery([]byte("((unknown) name: (identifier))"), getTestGrammar())
	if q != nil {
		t.Fatal("Expected q to be nil, got", q)
	}

	if err == nil {
		t.Fatal("Expected error to not be nil")
	}

	exp := DetailedQueryError{
		Offset:  2,
		Type:    QueryErrorNodeType,
		Message: "invalid node type 'unknown' at line 1 column 0",
	}
	if err.Error() != exp.Error() {
		t.Fatal("Error is not the expected QueryError:", err)
	}
}

func TestParserLifetime(t *testing.T) {
	t.Parallel()

	n, wg := 10, new(sync.WaitGroup)
	errs := make([]error, n*n)

	for i := range n {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := range n {
				p := NewParser()
				p.SetLanguage(getTestGrammar())

				data := []byte("1 + 2")
				// create some memory/CPU pressure
				data = append(data, bytes.Repeat([]byte(" "), 1024*1024)...)

				tree, err := p.ParseString(context.Background(), nil, data)
				if err != nil {
					errs[i*n+j] = err
					return
				}

				root := tree.RootNode()

				// must be a separate function, and it shouldn't accept the parser, only the Tree
				doWorkLifetime(t, root)
			}
		}()
	}

	wg.Wait()

	if err := cmp.Or(errs...); err != nil {
		t.Fatal(err)
	}
}

func TestTreeCursor(t *testing.T) { //nolint:tparallel // we test a specific navigation sequence
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

	if c.CurrentFieldName() != "" {
		t.Fatal("Expected current field name to be empty")
	}

	var nodeForReset *Node

	firstChild100 := func() bool { return c.GoToFirstChildForByte(100) == -1 }
	captureNodeForReset := func() bool { nodeForReset = c.CurrentNode(); return true }
	firstChild4 := func() bool { return c.GoToFirstChildForByte(4) == 2 }
	reset := func() bool { c.Reset(nodeForReset); return true }
	testCases := []struct {
		fn               func() bool
		exp              bool
		expType, expName string
	}{
		{c.GoToParent, false, "expression", ""},
		{c.GoToNextSibling, false, "expression", ""},
		{firstChild100, true, "expression", ""},
		{c.GoToFirstChild, true, "sum", ""},
		{c.GoToFirstChild, true, "expression", "left"},
		{c.GoToNextSibling, true, "+", ""},
		{c.GoToFirstChild, false, "+", ""},
		{c.GoToNextSibling, true, "expression", "right"},
		{c.GoToFirstChild, true, "number", ""},
		{c.GoToParent, true, "expression", "right"},
		{c.GoToParent, true, "sum", ""},
		// capture node for reset, not an actual test case
		{captureNodeForReset, true, "sum", ""},
		{firstChild4, true, "expression", "right"},
		// reset, not an actual case
		{reset, true, "sum", ""},
		{c.GoToParent, false, "sum", ""},
	}

	for _, tc := range testCases { //nolint:paralleltest // not applicable, see function level comment
		label := nameOf(tc.fn)

		t.Run(label, func(t *testing.T) {
			if act := tc.fn(); act != tc.exp {
				t.Fatalf("Expected c.%s() == %v, got %v", label, tc.exp, act)
			}

			if act := c.CurrentNode().Type(); act != tc.expType {
				t.Fatalf("Expected current node type to be %q, got %q", tc.expType, act)
			}

			if act := c.CurrentFieldName(); act != tc.expName {
				t.Fatalf("Expected current field name to be %q, got %q", tc.expName, act)
			}
		})
	}
}

func TestLeakParse(t *testing.T) {
	t.Parallel()

	parser := NewParser()
	parser.SetLanguage(getTestGrammar())

	for range 100_000 {
		parser.ParseString(context.Background(), nil, []byte("1 + 2")) //nolint:errcheck // ok
	}

	runtime.GC()

	var m runtime.MemStats

	runtime.ReadMemStats(&m)

	// Shouldn't exceed leakLimit that go runtime takes.
	// Was increased from upstream as we run tests in parallel.
	if x := m.Alloc; x >= leakLimit {
		t.Fatalf("Expected to only allocate %d, got %d", leakLimit, x)
	}
}

func TestLeakRootNode(t *testing.T) {
	t.Parallel()

	parser := NewParser()
	parser.SetLanguage(getTestGrammar())

	for range 100_000 {
		tree, err := parser.ParseString(context.Background(), nil, []byte("1 + 2"))
		if err != nil {
			t.Fatal("Expected no error, got", err)
		}

		tree.RootNode()
	}

	runtime.GC()

	var m runtime.MemStats

	runtime.ReadMemStats(&m)

	// Shouldn't exceed leakLimit go runtime takes.
	// Was increased from upstream as we run tests in parallel.
	if x := m.Alloc; x >= leakLimit {
		t.Fatalf("Expected to only allocate %d, got %d", leakLimit, x)
	}
}

func TestParseInput(t *testing.T) {
	t.Parallel()

	parser := NewParser()
	parser.SetLanguage(getTestGrammar())

	// empty input
	input := Input{
		Encoding: InputEncodingUTF8,
		Read:     func(_ uint32, _ Point) []byte { return nil },
	}

	tree, err := parser.Parse(context.Background(), nil, input)
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	n := tree.RootNode()
	exp := "(ERROR)"

	if x := n.String(); x != exp {
		t.Fatalf("Expected %q got %q", exp, x)
	}

	// return all data in one go
	readTimes := 0
	inputData := []byte("12345 + 23456")
	input.Read = func(_ uint32, _ Point) []byte {
		if readTimes > 0 {
			return nil
		}

		readTimes++

		return inputData
	}

	tree, err = parser.Parse(context.Background(), nil, input)
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	n = tree.RootNode()

	if x := n.String(); x != exprSumLR {
		t.Fatalf("Expected %q got %q", exprSumLR, x)
	}

	if readTimes != 1 {
		t.Fatal("Expected readTimes to be 1, got", readTimes)
	}

	// return all data in multiple sequantial reads
	input.Read = func(offset uint32, _ Point) []byte {
		if int(offset) >= len(inputData) {
			return nil
		}

		readTimes++

		end := int(offset + 5)
		if len(inputData) < end {
			end = len(inputData)
		}

		return inputData[offset:end]
	}

	tree, err = parser.Parse(context.Background(), nil, input)
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	n = tree.RootNode()

	if x := n.String(); x != exprSumLR {
		t.Fatalf("Expected %q got %q", exprSumLR, x)
	}

	if readTimes != 4 {
		t.Fatal("Expected readTimes to be 4, got", readTimes)
	}
}

func TestLeakParseInput(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	parser := NewParser()
	parser.SetLanguage(getTestGrammar())

	inputData := []byte("1 + 2")
	input := Input{
		Encoding: InputEncodingUTF8,
		Read: func(offset uint32, _ Point) []byte {
			if offset > 0 {
				return nil
			}

			return inputData
		},
	}

	for range 100_000 {
		parser.Parse(ctx, nil, input) //nolint:errcheck // ok
	}

	runtime.GC()

	var m runtime.MemStats

	runtime.ReadMemStats(&m)

	// Shouldn't exceed leakLimit that go runtime takes.
	// Was increased from upstream as we run tests in parallel.
	if x := m.Alloc; x >= leakLimit {
		t.Fatalf("Expected to only allocate %d, got %d", leakLimit, x)
	}
}

// see https://github.com/smacker/go-tree-sitter/issues/75
func TestCursorKeepsQuery(t *testing.T) {
	t.Parallel()

	source := bytes.Repeat([]byte("1 + 1"), 10000)

	parser := NewParser()
	parser.SetLanguage(getTestGrammar())

	tree, err := parser.ParseString(context.TODO(), nil, source)
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	root := tree.RootNode()

	for range 100 {
		query, _ := NewQuery( //nolint:errcheck // ok
			[]byte("(number) @match"),
			getTestGrammar(),
		)

		qc := NewQueryCursor()

		qc.Exec(query, root)

		for { // ensure qc.NextMatch() doesn't  cause a segfault
			match, exists := qc.NextMatch()
			if !exists || match == nil {
				break
			}
		}
	}
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

func BenchmarkParse(b *testing.B) {
	parser := NewParser()
	parser.SetLanguage(getTestGrammar())

	inputData := []byte("1 + 2")

	b.ResetTimer()

	for range b.N {
		parser.ParseString(context.Background(), nil, inputData) //nolint:errcheck // ok
	}
}

func BenchmarkParseCancellable(b *testing.B) {
	parser := NewParser()
	parser.SetLanguage(getTestGrammar())

	inputData := []byte("1 + 2")

	b.ResetTimer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for range b.N {
		parser.ParseString(ctx, nil, inputData) //nolint:errcheck // ok
	}
}

func BenchmarkParseInput(b *testing.B) {
	parser := NewParser()
	parser.SetLanguage(getTestGrammar())

	inputData := []byte("1 + 2")
	input := Input{
		Encoding: InputEncodingUTF8,
		Read: func(offset uint32, _ Point) []byte {
			if offset > 0 {
				return nil
			}

			return inputData
		},
	}

	b.ResetTimer()

	for range b.N {
		parser.Parse(context.Background(), nil, input) //nolint:errcheck // ok
	}
}

func testStartEnd(t *testing.T, n *Node, startByte, endByte, startCol, startRow, endRow, endCol uint32) {
	t.Helper()

	if x := n.StartByte(); x != startByte {
		t.Fatalf("Expected StartByte to be %d, got %d", startByte, x)
	}

	if x := n.EndByte(); x != endByte {
		t.Fatalf("Expected EndByte to be %d, got %d", endByte, x)
	}

	expPoint := Point{Column: startCol, Row: startRow}
	if x := n.StartPoint(); x != expPoint {
		t.Fatalf("Expected StartPoint to be %v, got %v", expPoint, x)
	}

	expPoint = Point{Column: endCol, Row: endRow}
	if x := n.EndPoint(); x != expPoint {
		t.Fatalf("Expected EndPoint to be %v, got %v", expPoint, x)
	}
}

func testCaptures(t *testing.T, body, sq string, exp []string) {
	t.Helper()

	parser := NewParser()
	parser.SetLanguage(getTestGrammar())

	tree, err := parser.ParseString(context.Background(), nil, []byte(body))
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	root := tree.RootNode()

	q, err := NewQuery([]byte(sq), getTestGrammar())
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}

	qc := NewQueryCursor()
	qc.Exec(q, root)

	act := []string{}

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		for _, c := range m.Captures {
			act = append(act, c.Node.Content([]byte(body)))
		}
	}

	if !slices.Equal(exp, act) {
		t.Fatalf("Expected %v, got %v", exp, act)
	}
}

func isNamedWithGC(n *Node) bool {
	runtime.GC()
	time.Sleep(500 * time.Microsecond)

	return n.IsNamed()
}

func doWorkLifetime(tb testing.TB, n *Node) {
	tb.Helper()

	for range 100 {
		if s := n.String(); s != exprSumLR {
			tb.Fatalf("Expected %q, got %q", exprSumLR, s)
		}
	}
}

func nameOf(fn any) (s string) {
	s = runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	tokens := strings.Split(s, ".")
	s = tokens[len(tokens)-1]
	tokens = strings.Split(s, "-")
	s = tokens[0]

	return
}
