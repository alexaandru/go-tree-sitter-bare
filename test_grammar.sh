# transforms grammar js file into go
# cgo can't be used in tests
out=test_grammar.go

export PATH=$PATH:./node_modules/.bin

echo Using $(tree-sitter --version)

tree-sitter generate ./test_grammar.js

echo "//go:build test" > $out
echo >> $out
echo "// Code generated by $0; DO NOT EDIT." >> $out
echo "package sitter" >> $out
echo >> $out
sed -e 's/^/\/\//' src/tree_sitter/parser.h >> $out
sed -e 's/^/\/\//' src/parser.c | grep -v '#include "tree_sitter/parser.h"' >> $out
cat <<GO >> $out
import "C"
import "unsafe"

func getTestGrammar() *Language {
	return NewLanguage(unsafe.Pointer(C.tree_sitter_test_grammar()))
}
GO

# cleanup
rm -rf *.toml setup.py grammar.js .editorconfig Package.swift binding.gyp build/ node_modules/ src/ bindings/
git restore Makefile .gitignore
