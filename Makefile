GOFLAGS ?= -tags=test
SHELL = /bin/bash

all: unimplemented todo fmt lint test

test:
	@GOFLAGS="$(GOFLAGS)" go test -cover -coverprofile=unit.cov .

check_lint:
	@golangci-lint version > /dev/null 2>&1 || \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint: check_lint
	@GOFLAGS="$(GOFLAGS)" golangci-lint run . && echo -e "ok\tno linter warnings"

fmt:
	@ls -1 *.go|while read x; do gofumpt -w -extra $$x; done

todo:
	@grep -E '(FIXME|TODO)(\s|:|"|$$)' *.go || true

# Show missing/unimplemented identifiers,
# except for wasm (ignored for now).
unimplemented:
	@comm -23 \
		<(grep ts_ api.h|grep -v '^ \*'|cut -f1 -d'('|sed -r 's/^(void|bool|uint32_t|const.*|TSQueryCursor|size_t|TS.*|u?int64_t|char) \*?//'|grep -v function|sort -u|grep -v wasm|grep -v ts_parser_parse|grep -v ts_set_allocator) \
		<(grep C.ts_ *.go|sed -r 's/^.*C.(ts_[0-9a-zA-Z_]*)\(.*$$/\1/g'|sort -u)

check_unimplemented:
	@[ -z "$$(make -s unimplemented)" ] && exit 0 || ( \
		make -s unimplemented| while read x; do echo "::notice title=API Not Implemented::$${x}"; done; exit 1)
