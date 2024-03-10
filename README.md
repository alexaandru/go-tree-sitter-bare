# Go Tree-Sitter Bare

## About this fork

This is a "fork" of @smacker's [go-tree-sitter](https://github.com/smacker/go-tree-sitter) as follows:

### Why

I needed strictly the sitter functionality without any of the parsers.
I needed this dependency to be as light as possible and had no need for
the parsers as I have created [my own repo](https://github.com/alexaandru/go-sitter-forest)
(also based on @smacker's work, but I needed a lot more parsers) and that
repo is itself 1.3GB already, so I didn't need another 200MB dependency
on sitter, when the actual code is less than 2MB.

### How

I copied the files from the root of `go-tree-sitter` plus my [PR](https://github.com/smacker/go-tree-sitter/pull/150),
as it was at `1f283e24f56023537e6de2d542732445e505f901` commit.
I kept the LICENSE and the README intact (except for this explanation of
why this fork exists) so although there is no git history, being a brand new
repo and not a GitHub fork, everything points back to the original author.

So there it is, full transparency and giving credit where is due. I generally
dislike deleting history, but simple `git rm`ing parsers from a clone, would've
still kept them in git history and contribute to the repo size.

## Original README continues below, unmodified

[![Build Status](https://github.com/smacker/go-tree-sitter/workflows/Test/badge.svg?branch=master)](https://github.com/smacker/go-tree-sitter/actions/workflows/test.yml?query=branch%3Amaster)
[![GoDoc](https://godoc.org/github.com/smacker/go-tree-sitter?status.svg)](https://godoc.org/github.com/smacker/go-tree-sitter)

Golang bindings for [tree-sitter](https://github.com/tree-sitter/tree-sitter)

## Usage

Create a parser with a grammar:

```go
import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
)

parser := sitter.NewParser()
parser.SetLanguage(javascript.GetLanguage())
```

Parse some code:

```go
sourceCode := []byte("let a = 1")
tree, _ := parser.ParseCtx(context.Background(), nil, sourceCode)
```

Inspect the syntax tree:

```go
n := tree.RootNode()

fmt.Println(n) // (program (lexical_declaration (variable_declarator (identifier) (number))))

child := n.NamedChild(0)
fmt.Println(child.Type()) // lexical_declaration
fmt.Println(child.StartByte()) // 0
fmt.Println(child.EndByte()) // 9
```

### Custom grammars

This repository provides grammars for many common languages out of the box.

But if you need support for any other language you can keep it inside your own project or publish it as a separate repository to share with the community. 

See explanation on how to create a grammar for go-tree-sitter [here](https://github.com/smacker/go-tree-sitter/issues/57).

Known external grammars:

- [Salesforce grammars](https://github.com/aheber/tree-sitter-sfapex) - including Apex, SOQL, and SOSL languages.
- [Ruby](https://github.com/shagabutdinov/go-tree-sitter-ruby) - Deprecated, grammar is provided by main repo instead
- [Go Template](https://github.com/mrjosh/helm-ls/tree/master/internal/tree-sitter/gotemplate) - Used for helm

### Editing

If your source code changes, you can update the syntax tree. This will take less time than the first parse.

```go
// change 1 -> true
newText := []byte("let a = true")
tree.Edit(sitter.EditInput{
    StartIndex:  8,
    OldEndIndex: 9,
    NewEndIndex: 12,
    StartPoint: sitter.Point{
        Row:    0,
        Column: 8,
    },
    OldEndPoint: sitter.Point{
        Row:    0,
        Column: 9,
    },
    NewEndPoint: sitter.Point{
        Row:    0,
        Column: 12,
    },
})

// check that it changed tree
assert.True(n.HasChanges())
assert.True(n.Child(0).HasChanges())
assert.False(n.Child(0).Child(0).HasChanges()) // left side of the tree didn't change
assert.True(n.Child(0).Child(1).HasChanges())

// generate new tree
newTree := parser.Parse(tree, newText)
```

### Predicates

You can filter AST by using [predicate](https://tree-sitter.github.io/tree-sitter/using-parsers#predicates) S-expressions.

Similar to [Rust](https://github.com/tree-sitter/tree-sitter/tree/master/lib/binding_rust) or [WebAssembly](https://github.com/tree-sitter/tree-sitter/blob/master/lib/binding_web) bindings we support filtering on a few common predicates:
- `eq?`, `not-eq?`
- `match?`, `not-match?`

Usage [example](./_examples/predicates/main.go):

```go
func main() {
	// Javascript code
	sourceCode := []byte(`
		const camelCaseConst = 1;
		const SCREAMING_SNAKE_CASE_CONST = 2;
		const lower_snake_case_const = 3;`)
	// Query with predicates
	screamingSnakeCasePattern := `(
		(identifier) @constant
		(#match? @constant "^[A-Z][A-Z_]+")
	)`

	// Parse source code
	lang := javascript.GetLanguage()
	n, _ := sitter.ParseCtx(context.Background(), sourceCode, lang)
	// Execute the query
	q, _ := sitter.NewQuery([]byte(screamingSnakeCasePattern), lang)
	qc := sitter.NewQueryCursor()
	qc.Exec(q, n)
	// Iterate over query results
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		// Apply predicates filtering
		m = qc.FilterPredicates(m, sourceCode)
		for _, c := range m.Captures {
			fmt.Println(c.Node.Content(sourceCode))
		}
	}
}

// Output of this program:
// SCREAMING_SNAKE_CASE_CONST
```

## Development

### Updating a grammar

Check if any updates for vendored files are available:

```
go run _automation/main.go check-updates
```

Update vendor files:

- open `_automation/grammars.json`
- modify `reference` (for tagged grammars) or `revision` (for grammars from a branch)
- run `go run _automation/main.go update <grammar-name>`

It is also possible to update all grammars in one go using

```
go run _automation/main.go update-all
```
