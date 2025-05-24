GOFLAGS ?= -tags=test
SHELL = /bin/bash

all: unimplemented todo fmt lint deadcode vulncheck test

test:
	@GOEXPERIMENT=cgocheck2 GOFLAGS="$(GOFLAGS)" go test -vet=all -race -cover -coverprofile=unit.cov .

lint:
	@GOFLAGS="$(GOFLAGS)" go tool -modfile=tools/go.mod golangci-lint config verify
	@GOFLAGS="$(GOFLAGS)" go tool -modfile=tools/go.mod golangci-lint run

deadcode:
	@go tool -modfile=tools/go.mod deadcode -tags=test -test ./...

vulncheck:
	@go tool -modfile=tools/go.mod govulncheck ./...

fmt:
	@find -name "*.go"|xargs go tool -modfile=tools/go.mod gofumpt -extra -w
	@find -name "*.go"|xargs go tool -modfile=tools/go.mod goimports -w

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
