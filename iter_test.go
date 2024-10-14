package sitter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"testing"
)

//nolint:gochecknoglobals,lll // ok
var (
	src1, src2 = "1 + 2", "((1 + 2) + (3 + 4))"

	iterTestCases = []struct {
		mode   IterMode
		source string
		exp    []string
	}{
		{DFS, src1, []string{"1 + 2 [expression]", "1 + 2 [sum]", "1 [expression]", "1 [number]", "+ [+]", "2 [expression]", "2 [number]"}},
		{BFS, src1, []string{"1 + 2 [expression]", "1 + 2 [sum]", "1 [expression]", "+ [+]", "2 [expression]", "1 [number]", "2 [number]"}},
		{DFSNamed, src1, []string{"1 + 2 [expression]", "1 + 2 [sum]", "1 [expression]", "1 [number]", "2 [expression]", "2 [number]"}},
		{BFSNamed, src1, []string{"1 + 2 [expression]", "1 + 2 [sum]", "1 [expression]", "2 [expression]", "1 [number]", "2 [number]"}},
		{DFS, src2, []string{"((1 + 2) + (3 + 4)) [expression]", "( [(]", "(1 + 2) + (3 + 4) [expression]", "(1 + 2) + (3 + 4) [sum]", "(1 + 2) [expression]", "( [(]", "1 + 2 [expression]", "1 + 2 [sum]", "1 [expression]", "1 [number]", "+ [+]", "2 [expression]", "2 [number]", ") [)]", "+ [+]", "(3 + 4) [expression]", "( [(]", "3 + 4 [expression]", "3 + 4 [sum]", "3 [expression]", "3 [number]", "+ [+]", "4 [expression]", "4 [number]", ") [)]", ") [)]"}},
		{BFS, src2, []string{"((1 + 2) + (3 + 4)) [expression]", "( [(]", "(1 + 2) + (3 + 4) [expression]", ") [)]", "(1 + 2) + (3 + 4) [sum]", "(1 + 2) [expression]", "+ [+]", "(3 + 4) [expression]", "( [(]", "1 + 2 [expression]", ") [)]", "( [(]", "3 + 4 [expression]", ") [)]", "1 + 2 [sum]", "3 + 4 [sum]", "1 [expression]", "+ [+]", "2 [expression]", "3 [expression]", "+ [+]", "4 [expression]", "1 [number]", "2 [number]", "3 [number]", "4 [number]"}},
		{DFSNamed, src2, []string{"((1 + 2) + (3 + 4)) [expression]", "(1 + 2) + (3 + 4) [expression]", "(1 + 2) + (3 + 4) [sum]", "(1 + 2) [expression]", "1 + 2 [expression]", "1 + 2 [sum]", "1 [expression]", "1 [number]", "2 [expression]", "2 [number]", "(3 + 4) [expression]", "3 + 4 [expression]", "3 + 4 [sum]", "3 [expression]", "3 [number]", "4 [expression]", "4 [number]"}},
		{BFSNamed, src2, []string{"((1 + 2) + (3 + 4)) [expression]", "(1 + 2) + (3 + 4) [expression]", "(1 + 2) + (3 + 4) [sum]", "(1 + 2) [expression]", "(3 + 4) [expression]", "1 + 2 [expression]", "3 + 4 [expression]", "1 + 2 [sum]", "3 + 4 [sum]", "1 [expression]", "2 [expression]", "3 [expression]", "4 [expression]", "1 [number]", "2 [number]", "3 [number]", "4 [number]"}},
	}
)

func TestNewIterator(t *testing.T) {
	t.Parallel()
	t.Skip("tested implicitly")
}

func TestIteratorNext(t *testing.T) {
	t.Parallel()

	for _, tc := range iterTestCases {
		t.Run(fmt.Sprintf("%s/%s", tc.source, tc.mode), func(t *testing.T) {
			t.Parallel()

			input := []byte(tc.source)

			root, err := Parse(context.Background(), input, gr)
			if err != nil {
				t.Fatal("Expected no error, got", err)
			}

			act := []string{}
			ii := NewIterator(root, tc.mode)

			var nn Node

			for {
				nn, err = ii.Next()
				if err != nil || nn == zeroNode {
					break
				}

				act = append(act, fmt.Sprintf("%s [%s]", nn.Content(input), nn.Type()))
			}

			if !errors.Is(err, io.EOF) {
				t.Fatal(err)
			}

			if !slices.Equal(act, tc.exp) {
				t.Fatalf("Expected\n%#v, got\n%#v\n", tc.exp, act)
			}
		})
	}
}

func TestIteratorForEach(t *testing.T) {
	t.Parallel()

	for _, tc := range iterTestCases {
		t.Run(fmt.Sprintf("%s/%s", tc.source, tc.mode), func(t *testing.T) {
			t.Parallel()

			input := []byte(tc.source)

			root, err := Parse(context.Background(), input, gr)
			if err != nil {
				t.Fatal("Expected no error, got", err)
			}

			act := []string{}

			err = NewIterator(root, tc.mode).ForEach(func(nn Node) error {
				act = append(act, fmt.Sprintf("%s [%s]", nn.Content(input), nn.Type()))
				return nil
			})

			if !errors.Is(err, io.EOF) {
				t.Fatal(err)
			}

			if !slices.Equal(act, tc.exp) {
				t.Fatalf("Expected\n%#v, got\n%#v\n", tc.exp, act)
			}
		})
	}
}
