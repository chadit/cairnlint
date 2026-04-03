package nofortestfunc

// Bad: exports internals for testing.
func GetCacheForTest() string { // want `function GetCacheForTest exports internals for testing`
	return "cache"
}

// Bad: ForTesting suffix.
func ResetStateForTesting() {} // want `function ResetStateForTesting exports internals for testing`

// Good: normal function name.
func GetCache() string {
	return "cache"
}

// Good: "ForTest" in the middle of the name is fine.
func ForTestingPurposes() {}
