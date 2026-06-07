package mixedreceiver

type Counter struct{ n int }

// Value receiver first, then a pointer receiver on the same type.
func (c Counter) Value() int { return c.n }
func (c *Counter) Inc()      {} // want `type Counter has both value and pointer receivers`

// Not flagged: all pointer receivers.
type Good struct{}

func (g *Good) A() {}
func (g *Good) B() {}
