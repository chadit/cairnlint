package preferwggo

import "sync"

// BadAddGoDeferDone is the classic pre-Go-1.25 pattern that wg.Go replaces.
func BadAddGoDeferDone() {
	var wg sync.WaitGroup
	wg.Add(1) // want `wg\.Add\(1\) \+ go func\(\)\{ defer wg\.Done\(\); \.\.\. \}\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	go func() {
		defer wg.Done()
		// work
	}()
	wg.Wait()
}

// BadInLoop same pattern inside a loop body. Each iteration should flag.
func BadInLoop() {
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1) // want `wg\.Add\(1\) \+ go func\(\)\{ defer wg\.Done\(\); \.\.\. \}\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
		go func() {
			defer wg.Done()
			// work
		}()
	}
	wg.Wait()
}

// BadPointerReceiver works the same with a pointer-typed WaitGroup.
func BadPointerReceiver() {
	wg := &sync.WaitGroup{}
	wg.Add(1) // want `wg\.Add\(1\) \+ go func\(\)\{ defer wg\.Done\(\); \.\.\. \}\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	go func() {
		defer wg.Done()
	}()
	wg.Wait()
}

type server struct {
	wg sync.WaitGroup
}

// BadStructField matches when the WaitGroup is a field on a struct.
func BadStructField() {
	s := server{}
	s.wg.Add(1) // want `wg\.Add\(1\) \+ go func\(\)\{ defer wg\.Done\(\); \.\.\. \}\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	go func() {
		defer s.wg.Done()
	}()
	s.wg.Wait()
}

type worker struct {
	sync.WaitGroup
}

// BadEmbedded matches promoted Add/Done on an embedded WaitGroup.
func BadEmbedded() {
	var w worker
	w.Add(1) // want `wg\.Add\(1\) \+ go func\(\)\{ defer wg\.Done\(\); \.\.\. \}\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	go func() {
		defer w.Done()
	}()
	w.Wait()
}

// BadMapReceiver flags Add/Done routed through map-indexed WaitGroups.
func BadMapReceiver() {
	wgs := map[string]*sync.WaitGroup{"main": {}}
	wgs["main"].Add(1) // want `wg\.Add\(1\) \+ go func\(\)\{ defer wg\.Done\(\); \.\.\. \}\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	go func() {
		defer wgs["main"].Done()
	}()
	wgs["main"].Wait()
}

// GoodAlreadyUsesWGGo is the modern Go 1.25 form the analyzer recommends.
func GoodAlreadyUsesWGGo() {
	var wg sync.WaitGroup
	wg.Go(func() {
		// work
	})
	wg.Wait()
}

// GoodAddNotOne skips non-literal-1 Add calls because wg.Go only adds 1.
// Migrating wg.Add(5) + a fan-out of 5 goroutines is not a mechanical swap.
func GoodAddNotOne() {
	var wg sync.WaitGroup
	wg.Add(5)
	for range 5 {
		go func() {
			defer wg.Done()
		}()
	}
	wg.Wait()
}

// GoodDoneNotDeferred skips the case where Done is called at the end but
// not deferred. The explicit Done fires only on the success path, while
// wg.Go's deferred Done fires on every exit, so the behavior could differ
// if the body panics.
func GoodDoneNotDeferred() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		// work
		wg.Done()
	}()
	wg.Wait()
}

// GoodDeferNotFirst skips cases where defer wg.Done() is not the first
// statement. Something earlier might itself defer, affecting ordering.
func GoodDeferNotFirst() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer println("cleanup")
		defer wg.Done()
	}()
	wg.Wait()
}

// GoodAddNotAdjacent requires the go statement to immediately follow Add.
// If there is an intervening statement the rewrite may change timing, so
// we do not suggest the swap.
func GoodAddNotAdjacent() {
	var wg sync.WaitGroup
	wg.Add(1)
	println("pre-spawn work")
	go func() {
		defer wg.Done()
	}()
	wg.Wait()
}

// GoodDifferentWaitGroups skips when Add and the deferred Done target
// different WaitGroup variables.
func GoodDifferentWaitGroups() {
	var wg1, wg2 sync.WaitGroup
	wg1.Add(1)
	go func() {
		defer wg2.Done()
	}()
	wg1.Done()
	wg2.Wait()
}

// GoodNamedFunc skips when the go statement calls a named function rather
// than an inline func literal. We do not know whether the body calls Done.
func GoodNamedFunc() {
	var wg sync.WaitGroup
	wg.Add(1)
	go namedWorker(&wg)
	wg.Wait()
}

func namedWorker(wg *sync.WaitGroup) {
	defer wg.Done()
}

// GoodEmptyFuncBody skips when the go func body is empty. There is no
// Done to replace and wg.Add(1) alone is a different bug.
func GoodEmptyFuncBody() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {}()
	wg.Done()
	wg.Wait()
}
