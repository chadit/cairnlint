package wgaddbeforego

import "sync"

// BadAddBeforeGo demonstrates the double-count bug: Add(1) + Go() = counter
// incremented by 2 but only decremented by 1 when the goroutine finishes.
// Wait blocks forever.
func BadAddBeforeGo() {
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1) // want `wg.Add before wg.Go is redundant; WaitGroup.Go calls Add\(1\) internally, this double-counts and hangs Wait`
		wg.Go(func() {
			// work
		})
	}
	wg.Wait()
}

// BadAddBeforeGoPointer uses a pointer receiver for the WaitGroup.
func BadAddBeforeGoPointer() {
	wg := &sync.WaitGroup{}
	wg.Add(1) // want `wg.Add before wg.Go is redundant; WaitGroup.Go calls Add\(1\) internally, this double-counts and hangs Wait`
	wg.Go(func() {
		// work
	})
	wg.Wait()
}

// GoodGoOnly is the correct usage: wg.Go handles Add/Done internally.
func GoodGoOnly() {
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			// work
		})
	}
	wg.Wait()
}

// GoodAddWithManualGoroutine is the pre-Go-1.25 pattern: Add + go + Done.
// This is fine because there's no wg.Go call.
func GoodAddWithManualGoroutine() {
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// work
		}()
	}
	wg.Wait()
}

// GoodAddAndGoOnDifferentVars has Add on one WaitGroup and Go on another.
// Should not flag because they target different variables.
func GoodAddAndGoOnDifferentVars() {
	var wg1 sync.WaitGroup
	var wg2 sync.WaitGroup
	wg1.Add(1)
	wg2.Go(func() {
		// work
	})
	go func() {
		defer wg1.Done()
		// work
	}()
	wg1.Wait()
	wg2.Wait()
}

// GoodAddNotFollowedByGo has an Add call but the next statement is not Go.
func GoodAddNotFollowedByGo() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
	}()
	wg.Wait()
}

// BadSeparatedInLoop has Add and Go separated by another statement.
func BadSeparatedInLoop() {
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1) // want `wg.Add before wg.Go is redundant; WaitGroup.Go calls Add\(1\) internally, this double-counts and hangs Wait`
		println("separated")
		wg.Go(func() {
			// work
		})
	}
	wg.Wait()
}

// BadAddBeforeLoop has Add before a loop that calls Go.
func BadAddBeforeLoop() {
	var wg sync.WaitGroup
	wg.Add(1) // want `wg.Add before wg.Go is redundant; WaitGroup.Go calls Add\(1\) internally, this double-counts and hangs Wait`
	for range 10 {
		wg.Go(func() {
			// work
		})
	}
	wg.Wait()
}

// GoodAddBeforeGoMixed has an Add for a go call, followed by wg.Go.
// This is not redundant because Add pairs with the manual go call.
func GoodAddBeforeGoMixed() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
	}()
	wg.Go(func() {
		// work
	})
	wg.Wait()
}

// BadAddBeforeIfGo has Add before an if block that calls Go.
func BadAddBeforeIfGo() {
	var wg sync.WaitGroup
	wg.Add(1) // want `wg.Add before wg.Go is redundant; WaitGroup.Go calls Add\(1\) internally, this double-counts and hangs Wait`
	if true {
		wg.Go(func() {
			// work
		})
	}
	wg.Wait()
}

type server struct {
	wg sync.WaitGroup
}

// BadStructField detects the pattern on struct field receivers.
func BadStructField() {
	s := server{}
	s.wg.Add(1) // want `wg.Add before wg.Go is redundant; WaitGroup.Go calls Add\(1\) internally, this double-counts and hangs Wait`
	s.wg.Go(func() {
		// work
	})
	s.wg.Wait()
}

// BadMapReceiver detects the pattern on map-accessed WaitGroups.
func BadMapReceiver() {
	wgs := map[string]*sync.WaitGroup{"main": {}}
	wgs["main"].Add(1) // want `wg.Add before wg.Go is redundant; WaitGroup.Go calls Add\(1\) internally, this double-counts and hangs Wait`
	wgs["main"].Go(func() {
		// work
	})
	wgs["main"].Wait()
}

// GoodMapDifferentKeys has Add and Go on different map keys.
func GoodMapDifferentKeys() {
	wgs := map[string]*sync.WaitGroup{"a": {}, "b": {}}
	wgs["a"].Add(1)
	wgs["b"].Go(func() {
		// work
	})
	go func() {
		defer wgs["a"].Done()
	}()
	wgs["a"].Wait()
	wgs["b"].Wait()
}

// BadSliceReceiver detects the pattern on slice-indexed WaitGroups.
func BadSliceReceiver() {
	wgs := []*sync.WaitGroup{{}}
	wgs[0].Add(1) // want `wg.Add before wg.Go is redundant; WaitGroup.Go calls Add\(1\) internally, this double-counts and hangs Wait`
	wgs[0].Go(func() {
		// work
	})
	wgs[0].Wait()
}
