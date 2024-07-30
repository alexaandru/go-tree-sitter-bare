GOFLAGS ?= -tags=test

all: lint test

test:
	@GOFLAGS="$(GOFLAGS)" go test -cover -coverprofile=unit.cov .

check_lint:
	@golangci-lint version > /dev/null 2>&1 || \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint: check_lint
	@GOFLAGS="$(GOFLAGS)" golangci-lint run . && echo -e "ok\tno linter warnings"

# Show missing/unimplemented identifiers,
# except for wasm and lookahead which are pending.
unimplemented_identifiers:
	@comm -23 \
		<(grep ts_ api.h|grep -v '^ \*'|cut -f1 -d'('|sed -r 's/^(void|bool|uint32_t|const.*|TSQueryCursor|size_t|TS.*|u?int64_t|char) \*?//'|grep -v function|sort -u|grep -v wasm|grep -v ts_lookahead) \
		<(grep C.ts_ *.go|sed -r 's/^.*C.(ts_[0-9a-zA-Z_]*)\(.*$$/\1/g'|sort -u)

