package preferbloop_test

import "testing"

func BenchmarkCStyleBN(b *testing.B) {
	for idx := 0; idx < b.N; idx++ { // want `use b\.Loop\(\) \{ \.\.\. \} instead of manual b\.N loop \(Go 1\.24\+\)`
		_ = idx
	}
}

func BenchmarkRangeBN(b *testing.B) {
	for range b.N { // want `use b\.Loop\(\) \{ \.\.\. \} instead of manual b\.N loop \(Go 1\.24\+\)`
		_ = "work"
	}
}

func BenchmarkBLoop(b *testing.B) {
	for b.Loop() {
		_ = "work"
	}
}

func BenchmarkUnrelatedLoop(b *testing.B) {
	items := []string{"a", "b", "c"}
	for _, item := range items {
		_ = item
	}
}

// b.Loop keeps its own timing window, so these two cannot be converted by
// dropping the b.N loop. See golang/go#74967.

func BenchmarkStopTimerAroundSetup(b *testing.B) {
	for idx := 0; idx < b.N; idx++ {
		b.StopTimer()

		payload := []byte("expensive setup")

		b.StartTimer()

		_ = len(payload)
	}
}

func BenchmarkStopTimerOutsideLoop(b *testing.B) {
	b.StopTimer()

	payload := []byte("expensive setup")

	b.StartTimer()

	for range b.N {
		_ = len(payload)
	}
}

// A sub-benchmark gets its own *testing.B, so timer calls in the parent do not
// exempt a loop inside the closure.
func BenchmarkSubBenchmarkStillFlagged(b *testing.B) {
	b.StopTimer()
	b.StartTimer()

	b.Run("inner", func(b *testing.B) {
		for range b.N { // want `use b\.Loop\(\) \{ \.\.\. \} instead of manual b\.N loop \(Go 1\.24\+\)`
			_ = "work"
		}
	})
}
