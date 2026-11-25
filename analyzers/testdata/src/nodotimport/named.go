package nodotimport

import (
	_ "embed"     // a blank import has a name, but not the dot
	str "strings" // a renamed import keeps the qualifier
)

var _ = str.ToUpper
