package nogetterprefix

type Config struct{}

// Flagged: zero-arg accessor with a Get prefix.
func (c Config) GetName() string { return "" } // want `accessor "GetName" has a Get prefix`

// Flagged: lowercase get prefix.
func getValue() int { return 0 } // want `accessor "getValue" has a Get prefix`

// Not flagged: no Get prefix.
func (c Config) Name() string { return "" }

// Not flagged: takes an argument, so it is a lookup, not a plain accessor.
func (c Config) GetByID(id int) string { return "" }

// Not flagged: Getenv is one word, not a Get prefix on Env.
func Getenv() string { return "" }

// Not flagged: Getter is one word.
func Getter() string { return "" }
