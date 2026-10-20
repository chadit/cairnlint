package stringconcatinloop

// prefix and joined are distinct package-level strings used in crossfile.go.
// Before type-based identity, both resolved to a nil ast.Object there.
var (
	prefix string
	joined string
)
