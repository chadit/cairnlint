// Package analyzers provides the analysis rules behind cairnlint. [All]
// returns the standard set and [AgentOnly] returns the heuristic agent tier.
// Each rule is a golang.org/x/tools/go/analysis analyzer with full type
// information. The README lists every analyzer.
//
// # Allocation in loops
//
// Three analyzers target repeated allocation inside loops, one of the most
// common Go performance mistakes. Whether it matters depends on data size.
//
// mapprealloc flags a map filled in a loop without a size hint. A map of 8 or
// fewer entries fits its first bucket, which can stay on the stack, and a hint
// of 9 or more forces a heap allocation (https://go.dev/issue/58214). Larger
// maps grow repeatedly without a hint.
//
// buildergrow flags a strings.Builder written in a loop without Grow.
// Doubling absorbs small outputs in two or three allocations, so Grow pays off
// above roughly 128 bytes of output.
//
// stringconcatinloop flags += on strings inside a loop. Each step copies the
// whole accumulated string, which is quadratic at any loop size, so this
// analyzer has no threshold.
//
// # Threshold configuration
//
// mapprealloc and buildergrow skip range loops over composite literals with
// fewer than min-range-len elements (default 8, the runtime's map group size).
// Other range sources are flagged, except that mapprealloc skips iterator
// functions and channels, which have no length to hint with. buildergrow
// also flags C-style for loops, whose iteration count is unknown.
//
//	cairnlint -mapprealloc.min-range-len=4 ./...
//	cairnlint -buildergrow.min-range-len=16 ./...
package analyzers
