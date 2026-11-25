//go:build go1.1

package modernize // Current: build constraints and an ordinary trailing comment.

// Current sits in a file with only the //go:build line, so plusbuild stays
// quiet, and its package clause carries a comment that is not an import
// path, so importcomment stays quiet too.
const Current = 2
