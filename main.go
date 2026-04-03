// cairnlint runs custom Go analysis rules that replace ruleguard
// and grep-based checks in lint.sh.
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/chadit/cairnlint/analyzers"
)

func main() {
	multichecker.Main(analyzers.All()...)
}
