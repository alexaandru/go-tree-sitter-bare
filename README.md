# Go Tree-Sitter bindings

![Build Status](https://github.com/alexaandru/go-tree-sitter-bare/actions/workflows/ci.yml/badge.svg)

**ONLY** provides the Go bindings for [tree-sitter](https://github.com/tree-sitter/tree-sitter).
For grammars see [go-sitter-forest](https://github.com/alexaandru/go-sitter-forest).

## Origins & Credits

This originally started as a "fork" of @smacker's [go-tree-sitter](https://github.com/smacker/go-tree-sitter),
read below to find out Why and How? It also recently adopted the
[query.go](https://github.com/tree-sitter/go-tree-sitter/blob/master/query.go)
from [go-tree-sitter](https://github.com/tree-sitter/go-tree-sitter)
so that we can filter captures/matches implicitly (no more need to
call `FilterPredicates()`).

### Why

I needed strictly the sitter functionality without any of the parsers and
I needed this dependency to be as light as possible and had no need for
the parsers as I have created [my own repo](https://github.com/alexaandru/go-sitter-forest)
(also based on @smacker's work, but I needed a lot more parsers) and that
repo is itself 1.4GB already, so I didn't need another 200MB dependency
on sitter, when the actual code is less than 2MB.

Technically, the [go-tree-sitter](https://github.com/tree-sitter/go-tree-sitter)
is now a viable alternative, in the sense that it does only offer the
bindings alone. Still, at 20MB in size is 20x times larger than this
repo. I may still keep my fork, we'll see...

### How

I copied the files from the root of `go-tree-sitter` plus my [PR](https://github.com/smacker/go-tree-sitter/pull/150),
as it was at `1f283e24f56023537e6de2d542732445e505f901` commit.
I kept the LICENSE so, although there is no git history, being a brand new
repo and not a GitHub fork, everything points back to the original author.

So there it is, full transparency and giving credit where is due. I generally
dislike deleting history, but simple `git rm`ing parsers from a clone, would've
still kept them in git history and contribute to the repo size.

### Differences

- timely kept up to date with `tree-sitter` updates (including new API calls);
- tiny, zero deps repo;
- implemented all API calls from [api.h](api.h) with the exception of WASM;
- reorganized code based on [api.h](api.h) sections and corresponding
  files (i.e. broken down `bindings.go` into [language.go](language.go),
  [parser.go](parser.go), [node.go](node.go), [query.go](query.go),
  [tree.go](tree.go), [tree_cursor.go](tree_cursor.go) and [sitter.go](sitter.go),
  with each file having the code sorted the same way as in `api.h`);
- synced **Go** funcs' comments with their counterparts from `api.h`;
- added return types where they were missing (a few places, where the C
  counterpart would return a bool but the Go wrapper wouldn't);
- some simplification related to types/params/etc. where it was possible;
- made all `Close()` methods private (as they are not supposed to be
  called by end users) and replaced their `isBool` with a `sync.Once`;
- added an automated check (and corresponding github action) to quickly
  check if we are falling behind [api.h](api.h);
- all tests run in parallel (so both faster and acting as an extra,
  indirect check that concurrency works as expected);
- code cleanup (enabled lots of linters and cleaned up code accordingly).

## Usage

Create a parser with a grammar:

```go
import (
	"context"
	"fmt"

	sitter "github.com/alexaandru/go-tree-sitter-bare"
	"github.com/alexaandru/go-sitter-forest/javascript"
)

var parser = sitter.NewParser()

func init() {
    if ok := parser.SetLanguage(sitter.NewLanguage(javascript.GetLanguage())); !ok {
        panic("cannot set language")
    }
}
```

Parse some code:

```go
sourceCode := []byte("let a = 1")
tree, err := parser.ParseString(context.Background(), nil, sourceCode)
if err != nil {
	panic(err)
}
defer tree.Close()

Inspect the syntax tree:

```go
n := tree.RootNode()

fmt.Println(n)
// (program (lexical_declaration (variable_declarator (identifier) (number))))

child := n.NamedChild(0)
fmt.Println(child.Type()) // lexical_declaration
fmt.Println(child.StartByte()) // 0
fmt.Println(child.EndByte()) // 9
```

### Editing

If your source code changes, you can update the syntax tree. This will take less time than the first parse.

```go
// change 1 -> true
newText := []byte("let a = true")
tree.Edit(sitter.EditInput{
    StartIndex: 8, OldEndIndex: 9, NewEndIndex: 12,
    StartPoint: sitter.Point{Row: 0, Column: 8},
    OldEndPoint: sitter.Point{Row: 0, Column: 9},
    NewEndPoint: sitter.Point{Row: 0, Column: 12},
})

// check that it changed tree
assert.True(n.HasChanges())
assert.True(n.Child(0).HasChanges())
assert.False(n.Child(0).Child(0).HasChanges()) // left side of the tree didn't change
assert.True(n.Child(0).Child(1).HasChanges())

// generate new tree
newTree, err := parser.ParseString(context.TODO(), tree, newText)
if err != nil {
	panic(err)
}
defer newTree.Close()
```

### Predicates

You can filter AST by using [predicate](https://tree-sitter.github.io/tree-sitter/using-parsers#predicates) S-expressions.

Similar to [Rust](https://github.com/tree-sitter/tree-sitter/tree/master/lib/binding_rust) or [WebAssembly](https://github.com/tree-sitter/tree-sitter/blob/master/lib/binding_web) bindings we support filtering on a few common predicates:

- `[any-][not-]eq?`,
- `[any-][not-]match?`,
- `[not-]any-of?`,
- `set?`,
- `is[-not]?`,
- and a default/catchall that simply saves the predicate op/args for later use,
  in `Query.generalPredicates`

Usage example:

```go
package main

import (
	"context"
	"fmt"

	"github.com/alexaandru/go-sitter-forest/javascript"
	sitter "github.com/alexaandru/go-tree-sitter-bare"
)

func main() {
	// Javascript code
	sourceCode := []byte(`
		const camelCaseConst = 1;
		const SCREAMING_SNAKE_CASE_CONST = 2;
		const lower_snake_case_const = 3;`)
	// Query with predicates
	query := []byte(`(
		(identifier) @constant
		(#match? @constant "^[A-Z][A-Z_]+"))`)

	// Get language.
	langPtr := javascript.GetLanguage()
	lang := sitter.NewLanguage(langPtr)

	// Parse source code.
	tree, err := parser.ParseString(context.Background(), nil, sourceCode)
	if err != nil {
		panic(err)
	}
	defer tree.Close()

	n := tree.RootNode()

	// Execute the query.
	q, _ := sitter.NewQuery(lang, query)
	qc := sitter.NewQueryCursor()

	matches := qc.Matches(q, n, sourceCode) // or q.Captures()

	for {
		m := matches.Next()
		if m == nil {
			break
		}

		for _, c := range m.Captures {
			fmt.Println(c.Node.Content(sourceCode))
			// Output: SCREAMING_SNAKE_CASE_CONST
		}
	}
}
```

## Resource Management and Tree Ownership

### Explicit Cleanup Required for Trees

As of Go 1.24+, resource management for Tree-sitter bindings is modernized. Most objects (Parser, Query, QueryCursor, TreeCursor) are cleaned up automatically using Go's `runtime.AddCleanup`. **However, Tree objects require explicit cleanup.**

- **You must call `tree.Close()` when you are done with a Tree.**
- Failing to call `Close()` will result in memory leaks.
- Using a Tree after calling `Close()` will panic or cause undefined behavior.
- This is necessary because Tree ownership is ambiguous and automatic cleanup could cause double-free or use-after-free bugs.

#### Example

```go
sourceCode := []byte("let a = 1")
tree, err := parser.ParseString(context.Background(), nil, sourceCode)
if err != nil {
	panic(err)
}
defer tree.Close() // Always close your trees!
```

#### Why not automatic cleanup for Tree?

Unlike other objects, Trees can be copied, edited, and passed around, making it impossible to safely determine ownership for automatic cleanup. Explicit `Close()` is the only safe approach.

#### Convenience Wrappers

If you want automatic cleanup, you can wrap Tree in your own type and use Go's `runtime.SetFinalizer` or similar, but this is not provided by default to avoid unsafe behavior.
