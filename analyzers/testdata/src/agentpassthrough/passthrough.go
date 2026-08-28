package agentpassthrough

import "strings"

type inner struct{ sep string }

// Join concatenates parts using the configured separator.
func (i *inner) Join(parts []string) string {
	return strings.Join(parts, i.sep)
}

// Sep reports the configured separator.
func (i *inner) Sep() string {
	return i.sep
}

// Wrapper holds one field and forwards every method to it.
type Wrapper struct { // want `every one of its 2 methods forwards`
	in *inner
}

// Join forwards to the wrapped joiner.
func (w *Wrapper) Join(parts []string) string {
	return w.in.Join(parts)
}

// Sep forwards to the wrapped joiner.
func (w *Wrapper) Sep() string {
	return w.in.Sep()
}

// Mixed does real work in one method, so only the forwarder is flagged.
type Mixed struct {
	in   *inner
	seen int
}

// Join forwards to the wrapped joiner.
func (m *Mixed) Join(parts []string) string { // want `only forwards its arguments to m\.in\.Join`
	return m.in.Join(parts)
}

// Count reports how many times it has been called.
func (m *Mixed) Count() int {
	m.seen++

	return m.seen
}

// Guarded adds a branch before delegating, so it is doing work.
type Guarded struct{ in *inner }

// Join returns the empty string for an empty input and delegates otherwise.
func (g *Guarded) Join(parts []string) string {
	if len(parts) == 0 {
		return ""
	}

	return g.in.Join(parts)
}

// Reordered changes the arguments on the way through, so it is not a forward.
type Reordered struct{ in *inner }

// Join delegates with the parts reversed.
func (r *Reordered) Join(parts []string) string {
	return r.in.Join(append([]string{"head"}, parts...))
}
