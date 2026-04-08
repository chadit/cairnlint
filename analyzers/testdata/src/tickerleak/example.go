package tickerleak

import "time"

// BadTickerNoStop leaks a goroutine because Stop is never called.
func BadTickerNoStop() {
	ticker := time.NewTicker(time.Second) // want `time\.NewTicker without defer Stop leaks a goroutine`
	_ = ticker
}

// BadTimerNoStop leaks until expiry because Stop is never called.
func BadTimerNoStop() {
	timer := time.NewTimer(time.Second) // want `time\.NewTimer without defer Stop leaks until expiry`
	_ = timer
}

// GoodTickerWithStop has proper cleanup.
func GoodTickerWithStop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	_ = ticker
}

// GoodTimerWithStop has proper cleanup.
func GoodTimerWithStop() {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	_ = timer
}

// GoodTickerReturned is a factory function; caller is responsible for Stop.
func GoodTickerReturned() *time.Ticker {
	ticker := time.NewTicker(time.Second)

	return ticker
}

// GoodTimerReturned is a factory function; caller is responsible for Stop.
func GoodTimerReturned() *time.Timer {
	timer := time.NewTimer(time.Second)

	return timer
}

type srv struct{ ticker *time.Ticker }

// GoodTickerAssignedToField stores the ticker in a struct for later cleanup.
func (s *srv) GoodTickerAssignedToField() {
	ticker := time.NewTicker(time.Second)
	s.ticker = ticker
}

// GoodStopInClosure has defer with Stop inside a closure wrapper.
func GoodStopInClosure() {
	ticker := time.NewTicker(time.Second)
	defer func() {
		ticker.Stop()
	}()
	_ = ticker
}
