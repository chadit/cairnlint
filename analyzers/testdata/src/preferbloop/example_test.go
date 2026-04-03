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
