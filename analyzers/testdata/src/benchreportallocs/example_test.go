package benchreportallocs

import "testing"

// BadBenchNoReportAllocs is missing ReportAllocs.
func BenchmarkBadNoReportAllocs(b *testing.B) { // want `benchmark missing b\.ReportAllocs\(\)`
	for b.Loop() {
		_ = make([]byte, 1024)
	}
}

// GoodBenchWithReportAllocs has ReportAllocs.
func BenchmarkGoodWithReportAllocs(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = make([]byte, 1024)
	}
}
