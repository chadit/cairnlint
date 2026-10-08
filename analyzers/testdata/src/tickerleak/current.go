//go:build go1.23

package tickerleak

import "time"

// CollectedTickerNoStop is not reported: from Go 1.23 the garbage collector
// reclaims an unreferenced ticker whether or not Stop was called.
func CollectedTickerNoStop() {
	ticker := time.NewTicker(time.Second)
	_ = ticker
}

// CollectedTimerNoStop is not reported for the same reason.
func CollectedTimerNoStop() {
	timer := time.NewTimer(time.Second)
	_ = timer
}
