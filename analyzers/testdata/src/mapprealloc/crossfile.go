package mapprealloc

// GoodSmallPackageVarOtherFile ranges over a three-element package-level
// slice declared in crossfile_decl.go, below the default minimum length.
func GoodSmallPackageVarOtherFile() map[string]int {
	index := map[string]int{}
	for idx, seed := range smallSeeds {
		index[seed] = idx
	}

	return index
}
