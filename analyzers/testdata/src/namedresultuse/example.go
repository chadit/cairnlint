package namedresultuse

import (
	"errors"
	"io"
	"os"
)

// Flagged: naked return relies on the named results.
func naked(p []byte) (n int, err error) {
	if len(p) == 0 {
		return // want `naked return in naked`
	}
	return len(p), nil
}

// Flagged: the named result is used as a working variable.
func assigned(p []byte) (n int, err error) {
	n = copy(p, "hello") // want `named result "n" is assigned in assigned`
	return n, nil
}

// Flagged once per result, not once per assignment.
func repeated(p []byte) (n int, err error) {
	n = 1 // want `named result "n" is assigned in repeated`
	n += 2
	n++
	err = errors.New("x") // want `named result "err" is assigned in repeated`
	return n, err
}

// Flagged: a range clause with = writes the named result.
func ranged(items []int) (last int) {
	for _, last = range items { // want `named result "last" is assigned in ranged`
	}
	return last
}

// Flagged: a non-deferred closure still writes the enclosing result.
func closureWrite() (count int) {
	func() {
		count = 3 // want `named result "count" is assigned in closureWrite`
	}()
	return count
}

// Flagged: function literals get the same treatment as declarations.
var lit = func() (n int) {
	n = 1 // want `named result "n" is assigned in func literal`
	return n
}

// Flagged: an op-assignment is still an assignment.
func opAssign() (total int) {
	total += 5 // want `named result "total" is assigned in opAssign`
	return total
}

// Not flagged: the names only document the signature, as io.Reader does.
func documented(p []byte) (n int, err error) {
	written := copy(p, "hello")
	return written, nil
}

// Not flagged: a deferred closure sets err, so the body has to share it.
func closeError(path string) (err error) {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	_, err = io.ReadAll(f)
	return err
}

// Not flagged: the recover idiom needs the named result too.
func recovers() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("recovered")
		}
	}()
	return nil
}

// Not flagged: the deferred closure reads the result but never writes it, and
// the body never assigns it either.
func deferReadOnly() (n int) {
	defer func() { _ = n }()
	return 1
}

// Not flagged: := declares a fresh variable and leaves the result alone.
func shadowedLocal() (n int) {
	if true {
		n := 2
		return n
	}
	return 1
}

// Not flagged: a field write mutates the pointee, not the result binding.
type box struct{ v int }

func fieldWrite() (b *box) {
	out := &box{}
	out.v = 1
	return out
}

// Not flagged: blank results cannot be used.
func blankResult() (_ int, err error) {
	return 1, nil
}

// Not flagged: no named results at all.
func unnamed() (int, error) {
	n := 1
	return n, nil
}

// Not flagged: interface signatures have no body to misuse the names in.
type Reader interface {
	Read(p []byte) (n int, err error)
}

// Not flagged: a naked return in a nested literal belongs to that literal,
// which has no named results of its own.
func nestedReturn() (n int) {
	f := func() {
		return
	}
	f()
	return 1
}
