# Show missing/unimplemented identifiers
unimplemented_identifiers:
	@comm -23 \
		<(grep ts_ api.h|grep -v '^ \*'|cut -f1 -d'('|sed -r 's/^(void|bool|uint32_t|const.*|TSQueryCursor|size_t|TS.*|u?int64_t|char) \*?//'|grep -v function|sort -u) \
		<(grep C.ts_ *.go|sed -r 's/^.*C.(ts_[0-9a-zA-Z_]*)\(.*$$/\1/g'|sort -u)

