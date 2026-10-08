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

// Under b.Loop the runner holds b.N at 1 and then 0, so a slice sized by b.N
// has one element and the converted loop indexes past it. Any b.N outside
// the loop condition keeps the benchmark as written.
func BenchmarkPresizedByBN(b *testing.B) {
	keys := make([]string, b.N)
	for idx := range keys {
		keys[idx] = "key"
	}

	for idx := 0; idx < b.N; idx++ {
		_ = keys[idx]
	}
}

// A second b.N loop would call b.Loop after the first finished, which fails
// with "B.Loop called with timer stopped".
func BenchmarkTwoBNLoops(b *testing.B) {
	for range b.N {
		_ = "first"
	}

	for range b.N {
		_ = "second"
	}
}

// A closure that captures the outer b is skipped even though it reads b.N
// once: the outer body sizes keys by b.N, which b.Loop holds at 1.
func BenchmarkClosureCapturesB(b *testing.B) {
	keys := make([]string, b.N)

	fill := func() {
		for idx := 0; idx < b.N; idx++ {
			keys[idx] = "key"
		}
	}

	fill()
}

// A goroutine body captures b the same way, so it is skipped too.
func BenchmarkGoroutineCapturesB(b *testing.B) {
	done := make(chan struct{})

	go func() {
		for range b.N {
			_ = "work"
		}

		close(done)
	}()

	<-done
}
