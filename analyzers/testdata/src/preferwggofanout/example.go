package preferwggofanout

import "sync"

// BadAddIdentRangeInt fires when Add(n) is followed by `for range n` and
// each iteration spawns a goroutine starting with defer wg.Done().
func BadAddIdentRangeInt() {
	var wg sync.WaitGroup

	n := 10

	wg.Add(n) // want `wg\.Add\(N\) \+ loop spawning N goroutines with defer wg\.Done\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	for range n {
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()
}

// BadAddLiteralRangeInt matches Add(10) + for range 10.
func BadAddLiteralRangeInt() {
	var wg sync.WaitGroup

	wg.Add(10) // want `wg\.Add\(N\) \+ loop spawning N goroutines with defer wg\.Done\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	for range 10 {
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()
}

// BadCStyleLoop matches Add(n) + canonical for i := 0; i < n; i++ form.
func BadCStyleLoop() {
	var wg sync.WaitGroup

	n := 5

	wg.Add(n) // want `wg\.Add\(N\) \+ loop spawning N goroutines with defer wg\.Done\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()
}

// BadLenSliceRange matches Add(len(items)) + `for range items` because
// both normalize to len(items) iterations.
func BadLenSliceRange() {
	var wg sync.WaitGroup

	items := []int{1, 2, 3}

	wg.Add(len(items)) // want `wg\.Add\(N\) \+ loop spawning N goroutines with defer wg\.Done\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	for range items {
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()
}

// BadLenSliceRangeWithVar matches when range declares vars used inside the
// goroutine. The defer is still the first statement, so wg.Go is valid.
func BadLenSliceRangeWithVar() {
	var wg sync.WaitGroup

	items := []int{1, 2, 3}

	wg.Add(len(items)) // want `wg\.Add\(N\) \+ loop spawning N goroutines with defer wg\.Done\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	for _, item := range items {
		go func(value int) {
			defer wg.Done()

			_ = value
		}(item)
	}

	wg.Wait()
}

// BadLenCStyleLoop matches Add(len(items)) + for i := 0; i < len(items); i++.
func BadLenCStyleLoop() {
	var wg sync.WaitGroup

	items := []int{1, 2, 3}

	wg.Add(len(items)) // want `wg\.Add\(N\) \+ loop spawning N goroutines with defer wg\.Done\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	for i := 0; i < len(items); i++ {
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()
}

// BadRangeLenInt matches Add(len(items)) + for range len(items). Both sides
// normalize to len(items), so this is also a fan-out candidate.
func BadRangeLenInt() {
	var wg sync.WaitGroup

	items := []int{1, 2, 3}

	wg.Add(len(items)) // want `wg\.Add\(N\) \+ loop spawning N goroutines with defer wg\.Done\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	for range len(items) {
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()
}

// BadPointerReceiver works the same with a pointer-typed WaitGroup.
func BadPointerReceiver() {
	wg := &sync.WaitGroup{}

	n := 4

	wg.Add(n) // want `wg\.Add\(N\) \+ loop spawning N goroutines with defer wg\.Done\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	for range n {
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()
}

type server struct {
	wg sync.WaitGroup
}

// BadStructField matches when the WaitGroup lives on a struct field.
func BadStructField() {
	s := server{}

	n := 3

	s.wg.Add(n) // want `wg\.Add\(N\) \+ loop spawning N goroutines with defer wg\.Done\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	for range n {
		go func() {
			defer s.wg.Done()
		}()
	}

	s.wg.Wait()
}

type worker struct {
	sync.WaitGroup
}

// BadEmbedded matches promoted Add/Done on an embedded WaitGroup.
func BadEmbedded() {
	var w worker

	n := 2

	w.Add(n) // want `wg\.Add\(N\) \+ loop spawning N goroutines with defer wg\.Done\(\) can be replaced with wg\.Go\(fn\) \(Go 1\.25\)`
	for range n {
		go func() {
			defer w.Done()
		}()
	}

	w.Wait()
}

// GoodAlreadyWGGo is the modern Go 1.25 form the analyzer recommends.
func GoodAlreadyWGGo() {
	var wg sync.WaitGroup

	for range 10 {
		wg.Go(func() {
			// work
		})
	}

	wg.Wait()
}

// GoodCountMismatch skips when Add count does not match the loop count.
// The bulk Add might be balanced by Done calls elsewhere.
func GoodCountMismatch() {
	var wg sync.WaitGroup

	wg.Add(10)

	for range 5 {
		go func() {
			defer wg.Done()
		}()
	}

	for range 5 {
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()
}

// GoodDifferentIdent skips when Add and the loop use different identifiers,
// even though they might hold the same value at runtime.
func GoodDifferentIdent() {
	var wg sync.WaitGroup

	n := 5
	m := 5

	wg.Add(n)

	for range m {
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()
}

// GoodNonAdjacent skips when an intervening statement separates Add from
// the fan-out loop, since the rewrite would re-order that work.
func GoodNonAdjacent() {
	var wg sync.WaitGroup

	n := 3

	wg.Add(n)
	println("pre-fanout work")

	for range n {
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()
}

// GoodDoneNotDeferred skips when Done is called explicitly at the end of
// the goroutine instead of deferred. Behavior differs on panic.
func GoodDoneNotDeferred() {
	var wg sync.WaitGroup

	n := 3

	wg.Add(n)

	for range n {
		go func() {
			// work
			wg.Done()
		}()
	}

	wg.Wait()
}

// GoodDeferNotFirst skips when defer wg.Done() is not the first statement.
// Something earlier might itself be a defer, affecting unwind order.
func GoodDeferNotFirst() {
	var wg sync.WaitGroup

	n := 3

	wg.Add(n)

	for range n {
		go func() {
			defer println("cleanup")
			defer wg.Done()
		}()
	}

	wg.Wait()
}

// GoodExtraStmtInLoop skips when the loop body has work alongside the go
// statement. wg.Go would force a choice between moving that work inside
// or outside the closure, either of which changes timing.
func GoodExtraStmtInLoop() {
	var wg sync.WaitGroup

	n := 3

	wg.Add(n)

	for i := range n {
		println("iter", i)
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()
}

// GoodDifferentReceivers skips when Add and Done target different WGs.
func GoodDifferentReceivers() {
	var wg1, wg2 sync.WaitGroup

	n := 3

	wg1.Add(n)

	for range n {
		go func() {
			defer wg2.Done()
		}()
	}

	for range n {
		wg1.Done()
	}

	wg2.Wait()
}

// GoodNamedFunc skips when the go statement calls a named function rather
// than an inline func literal. The body is opaque to the analyzer.
func GoodNamedFunc() {
	var wg sync.WaitGroup

	n := 3

	wg.Add(n)

	for range n {
		go namedWorker(&wg)
	}

	wg.Wait()
}

func namedWorker(wg *sync.WaitGroup) {
	defer wg.Done()
}

// GoodArithmeticAdd skips when Add uses arithmetic the analyzer can't
// compare reliably (BinaryExpr returns "" from countExprString).
func GoodArithmeticAdd() {
	var wg sync.WaitGroup

	n := 3

	wg.Add(n + 1)

	for range n + 1 {
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()
}

// GoodCStyleNonZeroStart skips C-style loops that don't start at zero,
// since the iteration count is not just the bound expression.
func GoodCStyleNonZeroStart() {
	var wg sync.WaitGroup

	n := 5

	wg.Add(n)

	for i := 1; i < n; i++ {
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()
}

// GoodCStyleLessOrEqual skips C-style loops using <= rather than <, since
// the iteration count is N+1, not N.
func GoodCStyleLessOrEqual() {
	var wg sync.WaitGroup

	n := 5

	wg.Add(n)

	for i := 0; i <= n; i++ {
		go func() {
			defer wg.Done()
		}()
	}

	wg.Wait()
}
