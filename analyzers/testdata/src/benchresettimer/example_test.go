package benchresettimer

import "testing"

// BenchmarkBadSetupNoReset has setup before a b.N loop but no ResetTimer.
func BenchmarkBadSetupNoReset(b *testing.B) { // want `benchmark has setup code without b\.ResetTimer\(\); setup time is included in measurements`
	data := make([]byte, 1024*1024) // setup
	for i := 0; i < b.N; i++ {
		_ = len(data)
	}
}

// BenchmarkBadSetupNoResetRange has setup before a range b.N loop but no ResetTimer.
func BenchmarkBadSetupNoResetRange(b *testing.B) { // want `benchmark has setup code without b\.ResetTimer\(\); setup time is included in measurements`
	data := make([]byte, 1024*1024) // setup
	for range b.N {
		_ = len(data)
	}
}

// BenchmarkGoodSetupWithReset has setup and ResetTimer before a b.N loop.
func BenchmarkGoodSetupWithReset(b *testing.B) {
	data := make([]byte, 1024*1024) // setup
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = len(data)
	}
}

// BenchmarkGoodNoSetup has no setup code.
func BenchmarkGoodNoSetup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = make([]byte, 64)
	}
}

// BenchmarkGoodLoopSetup has setup before b.Loop(), which resets the timer
// itself on its first call, so no ResetTimer is needed.
func BenchmarkGoodLoopSetup(b *testing.B) {
	data := make([]byte, 1024*1024) // setup
	for b.Loop() {
		_ = len(data)
	}
}
