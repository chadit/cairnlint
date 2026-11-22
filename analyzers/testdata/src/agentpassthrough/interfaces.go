package agentpassthrough

import (
	"agentpassthrough/api"
	"fmt"
	"io"
	"sort"
)

type engine struct{ reason string }

// Stop halts the engine.
func (e *engine) Stop() error {
	e.reason = "stop"

	return nil
}

// Halt halts the engine for a reason.
func (e *engine) Halt(reason string) error {
	e.reason = reason

	return nil
}

// Label names the engine.
func (e *engine) Label() string {
	if e.reason == "" {
		return "idle"
	}

	return e.reason
}

// Closer is declared in this package.
type Closer interface{ Close() error }

// samePkg satisfies Closer by forwarding Close to a differently named method.
type samePkg struct {
	inner *engine
	n     int
}

// Close is required by Closer, which engine does not implement.
func (s *samePkg) Close() error { return s.inner.Stop() }

// Count does real work.
func (s *samePkg) Count() int { s.n++; return s.n }

// otherPkg satisfies api.Stopper.
type otherPkg struct {
	inner *engine
	n     int
}

// Shutdown is required by api.Stopper.
func (o *otherPkg) Shutdown() error { return o.inner.Stop() }

// Count does real work.
func (o *otherPkg) Count() int { o.n++; return o.n }

// labeled satisfies fmt.Stringer, which only the standard library declares.
type labeled struct {
	inner *engine
	n     int
}

// String is required by fmt.Stringer.
func (l *labeled) String() string { return l.inner.Label() }

// Count does real work.
func (l *labeled) Count() int { l.n++; return l.n }

// HalterCloser is satisfied by forwarders alone.
type HalterCloser interface {
	Close() error
	Halt(reason string) error
}

// wrapper has one field and two required forwarders, so the type is not a
// middle man.
type wrapper struct{ inner *engine }

// Close is required by HalterCloser.
func (w *wrapper) Close() error { return w.inner.Stop() }

// Halt is required by HalterCloser, which engine does not implement.
func (w *wrapper) Halt(reason string) error { return w.inner.Halt(reason) }

// plain forwards a method no interface declares.
type plain struct {
	inner *engine
	n     int
}

// Stop forwards to the engine.
func (p *plain) Stop() error { return p.inner.Stop() } // want `^\[agent\] method plain\.Stop only forwards its arguments to p\.inner\.Stop`

// Count does real work.
func (p *plain) Count() int { p.n++; return p.n }

// fileCloser holds a value that is itself an io.Closer, so the holder can go.
type fileCloser struct {
	file io.Closer
	n    int
}

// Close forwards to a value that implements io.Closer too.
func (f *fileCloser) Close() error { return f.file.Close() } // want `^\[agent\] method fileCloser\.Close only forwards its arguments to f\.file\.Close`

// Count does real work.
func (f *fileCloser) Count() int { f.n++; return f.n }

// IntSorter is satisfied by a forwarder to a package-level function.
type IntSorter interface{ Ints(x []int) }

// pkgFwd forwards Ints to sort.Ints, which no value can stand in for.
type pkgFwd struct {
	n int
}

// Ints is required by IntSorter.
func (p *pkgFwd) Ints(x []int) { sort.Ints(x) }

// Count does real work.
func (p *pkgFwd) Count() int { p.n++; return p.n }

// solo has one field, one required forwarder, and one plain forwarder, so
// only the plain one is reported and the type is not.
type solo struct{ inner *engine }

// Close is required by Closer.
func (s *solo) Close() error { return s.inner.Stop() }

// Stop forwards to the engine.
func (s *solo) Stop() error { return s.inner.Stop() } // want `^\[agent\] method solo\.Stop only forwards its arguments to s\.inner\.Stop`

var (
	_ Closer       = (*samePkg)(nil)
	_ fmt.Stringer = (*labeled)(nil)
	_ api.Stopper  = (*otherPkg)(nil)
	_ HalterCloser = (*wrapper)(nil)
	_ io.Closer    = (*fileCloser)(nil)
	_ IntSorter    = (*pkgFwd)(nil)
	_ Closer       = (*solo)(nil)
)
