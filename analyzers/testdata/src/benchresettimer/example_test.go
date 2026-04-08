package benchresettimer

import "testing"

// BenchmarkBadSetupNoReset has setup but no ResetTimer.
func BenchmarkBadSetupNoReset(b *testing.B) { // want `benchmark has setup code without b\.ResetTimer\(\); setup time is included in measurements`
	data := make([]byte, 1024*1024) // setup
	_ = data
	for b.Loop() {
		_ = len(data)
	}
}

// BenchmarkGoodSetupWithReset has setup and ResetTimer.
func BenchmarkGoodSetupWithReset(b *testing.B) {
	data := make([]byte, 1024*1024) // setup
	b.ResetTimer()
	for b.Loop() {
		_ = len(data)
	}
}

// BenchmarkGoodNoSetup has no setup code.
func BenchmarkGoodNoSetup(b *testing.B) {
	for b.Loop() {
		_ = make([]byte, 64)
	}
}
