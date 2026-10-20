package stringconcatinloop

// GoodDistinctPackageVarsOtherFile assigns one string from another, so the
// loop never grows a single accumulator.
func GoodDistinctPackageVarsOtherFile(items []string) string {
	for _, item := range items {
		joined = prefix + item
	}

	return joined
}
