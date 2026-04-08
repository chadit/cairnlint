package wgdoneinwggo

import "sync"

// BadDeferDoneInGo has defer wg.Done() inside wg.Go - double decrement.
func BadDeferDoneInGo() {
	var wg sync.WaitGroup
	wg.Go(func() {
		defer wg.Done() // want `wg\.Done inside wg\.Go is redundant; WaitGroup\.Go calls Done automatically when f returns`
	})
	wg.Wait()
}

// BadDirectDoneInGo has wg.Done() without defer inside wg.Go.
func BadDirectDoneInGo() {
	var wg sync.WaitGroup
	wg.Go(func() {
		wg.Done() // want `wg\.Done inside wg\.Go is redundant; WaitGroup\.Go calls Done automatically when f returns`
	})
	wg.Wait()
}

// GoodGoNoDone is the correct usage of wg.Go.
func GoodGoNoDone() {
	var wg sync.WaitGroup
	wg.Go(func() {
		// work without Done
	})
	wg.Wait()
}

// GoodDifferentWGs has Done on a different WaitGroup than Go.
func GoodDifferentWGs() {
	var wg1 sync.WaitGroup
	var wg2 sync.WaitGroup
	wg2.Add(1)
	wg1.Go(func() {
		wg2.Done() // different WG, not flagged
	})
	wg1.Wait()
	wg2.Wait()
}

// GoodManualGoroutine uses the pre-1.25 pattern with go + Done.
func GoodManualGoroutine() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done() // manual goroutine, Done is correct here
	}()
	wg.Wait()
}

type server struct {
	wg sync.WaitGroup
}

// BadStructFieldDoneInGo has Done on struct field inside Go.
func BadStructFieldDoneInGo() {
	s := server{}
	s.wg.Go(func() {
		defer s.wg.Done() // want `wg\.Done inside wg\.Go is redundant; WaitGroup\.Go calls Done automatically when f returns`
	})
	s.wg.Wait()
}
