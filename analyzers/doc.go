// Package analyzers provides custom Go analysis rules for cairnlint.
// Each analyzer is registered via a constructor function in All() and
// uses the golang.org/x/tools/go/analysis framework for AST inspection
// with full type information.
//
// # Memory allocation analyzers
//
// Three analyzers target allocation patterns in loops: [mapprealloc],
// [buildergrow], and [stringconcatinloop]. They exist because repeated
// allocation inside a loop is one of the most common Go performance
// mistakes, but the right response depends on the data size.
//
// ## Map pre-allocation (mapprealloc)
//
// The Go runtime allocates one bucket (8 slots) when a map is created
// with no size hint. The load factor is 6.5 entries per bucket, so the
// first growth event triggers around 7 entries. When a map does not
// escape the function, the compiler can place that first bucket on the
// stack with zero heap allocation.
//
// Calling make(map[K]V, 9) or higher forces a heap allocation even when
// the map would otherwise stay on the stack (see https://go.dev/issue/58214).
// For maps that will hold 8 or fewer entries, adding a size hint offers
// no benefit and can make performance worse.
//
// For larger maps the cost grows quickly: a 10,000-entry map without
// pre-allocation uses roughly 625 allocations and is ~1.8x slower than
// one created with make(map[K]V, 10000).
//
// The analyzer skips diagnostics when the range source is a composite
// literal with fewer than min-range-len elements (default 8). Dynamic
// range sources are always flagged because the size is unknown.
//
// ## Builder Grow (buildergrow)
//
// strings.Builder starts with a nil internal buffer. Its growth strategy
// allocates 2*cap + n bytes on each expansion. For small iteration
// counts (under ~8) the doubling handles the work in 2-3 allocations
// regardless of whether Grow was called. The savings from Grow become
// meaningful above ~128 bytes of total output, where it prevents 4+
// growth events.
//
// Like mapprealloc, the analyzer skips diagnostics for range loops over
// small literal sources (below min-range-len, default 8). C-style for
// loops are always flagged because the iteration count cannot be
// determined statically.
//
// ## String concatenation in loops (stringconcatinloop)
//
// Go strings are immutable. Each += in a loop allocates a new backing
// array, copies the entire accumulated string, then appends the new
// piece. This is O(n^2) in both time and allocation regardless of the
// string sizes involved. At 1,000 iterations the cost is roughly 91x
// slower and 474x more allocations compared to strings.Builder.
//
// Because the quadratic cost applies at any loop size, this analyzer has
// no threshold and always flags concatenation inside loops.
//
// ## Threshold configuration
//
// Both mapprealloc and buildergrow accept a -min-range-len flag
// (default 8) that controls the element count below which diagnostics
// on literal range sources are suppressed. The default aligns with the
// Go runtime's map bucket size.
//
//	cairnlint -mapprealloc.min-range-len=4 ./...
//	cairnlint -buildergrow.min-range-len=16 ./...
//
// ## References
//
// Map internals and pre-allocation:
//
//   - Go blog, "Faster Go maps with Swiss Tables" (Go 1.24): https://go.dev/blog/swisstable
//   - Dave Cheney, "How the Go runtime implements maps efficiently": https://dave.cheney.net/2018/05/29/how-the-go-runtime-implements-maps-efficiently-without-generics
//   - Stack-allocated small map buckets (Go issue #58214): https://github.com/golang/go/issues/58214
//   - Effective Go, "Allocation with make": https://go.dev/doc/effective_go#allocation_make
//
// String performance:
//
//   - strings.Builder API: https://pkg.go.dev/strings#Builder
//   - strings.Builder source (growth strategy): https://go.dev/src/strings/builder.go
//
// General Go performance and pre-allocation best practices:
//
//   - Uber Go Style Guide, "Prefer Specifying Container Capacity": https://github.com/uber-go/guide/blob/master/style.md
//   - Dave Cheney, "High Performance Go Workshop" (GopherCon 2019): https://dave.cheney.net/high-performance-go-workshop/gophercon-2019.html
//   - Dave Cheney, "Don't force allocations on the callers of your API": https://dave.cheney.net/2019/09/05/dont-force-allocations-on-the-callers-of-your-api
package analyzers
