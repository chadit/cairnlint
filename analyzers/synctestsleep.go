package analyzers

import "golang.org/x/tools/go/analysis"

// synctestSleepAnalyzer returns an analyzer that flags time.Sleep calls
// in test files that are not inside a synctest.Test closure. Inside
// synctest.Test, time.Sleep advances synthetic time and is the correct
// API. Outside of it, time.Sleep causes flaky tests.
func synctestSleepAnalyzer() *analysis.Analyzer {
	return testCallWithSynctestExemption(synctestExemptConfig{
		name:     "synctestsleep",
		doc:      "flags time.Sleep in test files unless inside a synctest.Test closure",
		message:  "time.Sleep in tests is a flaky test signal; use synctest.Test with synthetic time, channels, or t.Deadline() instead",
		matchers: []callMatcher{{pkgPath: "time", funcName: "Sleep"}},
	})
}
