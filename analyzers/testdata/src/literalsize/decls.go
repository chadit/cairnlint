package literalsize

// File-scope const and var declarations are where named values live.
const (
	pageSize = 4096
	maxDepth = 8
)

var _ [1]struct{}

var (
	pool = make([]byte, 0, pageSize)
	sem  = make(chan struct{}, 4)
)

var table = []struct {
	name string
	size int
}{
	{name: "over limit", size: 1048577},
}
