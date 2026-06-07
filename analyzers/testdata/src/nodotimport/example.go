package nodotimport

import . "strings" // want `dot import "strings"; qualify the package explicitly instead`

// referenced so the dot import is used and the file compiles.
var _ = TrimSpace
