package gowggo

import "sync"

// BadGoWGGo wraps wg.Go with go keyword, racing Add with Wait.
func BadGoWGGo() {
	var wg sync.WaitGroup
	go wg.Go(func() {}) // want `go wg\.Go\(\.\.\.\) is a bug; wg\.Go calls Add\(1\) internally, wrapping with go races Add with Wait`
	wg.Wait()
}

// BadGoWGGoPointer same bug with pointer receiver.
func BadGoWGGoPointer() {
	wg := &sync.WaitGroup{}
	go wg.Go(func() {}) // want `go wg\.Go\(\.\.\.\) is a bug; wg\.Go calls Add\(1\) internally, wrapping with go races Add with Wait`
	wg.Wait()
}

// GoodWGGo is the correct usage without go keyword.
func GoodWGGo() {
	var wg sync.WaitGroup
	wg.Go(func() {})
	wg.Wait()
}

// GoodGoClosureWithWGGo wraps a closure that internally calls wg.Go.
// The go keyword starts the closure, not wg.Go directly.
func GoodGoClosureWithWGGo() {
	var wg sync.WaitGroup
	go func() {
		wg.Go(func() {})
	}()
	wg.Wait()
}

type worker struct {
	sync.WaitGroup
}

// BadGoWGGoEmbedded same bug with embedded WaitGroup.
func BadGoWGGoEmbedded() {
	var w worker
	go w.Go(func() {}) // want `go wg\.Go\(\.\.\.\) is a bug; wg\.Go calls Add\(1\) internally, wrapping with go races Add with Wait`
	w.Wait()
}
