// Code generated by "stringer -type=IterMode -output=stringer2.go ."; DO NOT EDIT.

package sitter

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[DFS-0]
	_ = x[BFS-1]
	_ = x[DFSNamed-2]
	_ = x[BFSNamed-3]
}

const _IterMode_name = "DFSBFSDFSNamedBFSNamed"

var _IterMode_index = [...]uint8{0, 3, 6, 14, 22}

func (i IterMode) String() string {
	if i < 0 || i >= IterMode(len(_IterMode_index)-1) {
		return "IterMode(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _IterMode_name[_IterMode_index[i]:_IterMode_index[i+1]]
}