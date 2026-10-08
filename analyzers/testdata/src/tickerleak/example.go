package tickerleak

import "time"

// An untagged file carries no language version in this fixture, which the
// gate reads as current Go, so nothing here is reported. legacy.go pins
// Go 1.22 for every case that exercises the cleanup detection.

// UntaggedTickerNoStop is not reported: no version means current Go.
func UntaggedTickerNoStop() {
	ticker := time.NewTicker(time.Second)
	_ = ticker
}
