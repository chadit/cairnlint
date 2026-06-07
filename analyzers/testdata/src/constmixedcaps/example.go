package constmixedcaps

// Flagged: underscore in a constant name.
const MAX_SIZE = 512 // want `constant "MAX_SIZE" contains an underscore`

const (
	Foo_Bar    = 1 // want `constant "Foo_Bar" contains an underscore`
	kMaxBuffer = 2 // want `constant "kMaxBuffer" uses a k-prefix`
	MaxSize    = 3
	maxLength  = 4
)

// Not flagged: an initialism with no underscore.
const URL = "https://example.com"
