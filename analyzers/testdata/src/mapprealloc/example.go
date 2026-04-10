package mapprealloc

// BadMapLiteralInLoop populates map literal in range loop.
func BadMapLiteralInLoop(items []string) map[string]int {
	index := map[string]int{} // want `map index populated in range loop without capacity hint`
	for idx, item := range items {
		index[item] = idx
	}

	return index
}

// BadMakeMapInLoop uses make without capacity.
func BadMakeMapInLoop(items []string) map[string]int {
	index := make(map[string]int) // want `map index populated in range loop without capacity hint`
	for idx, item := range items {
		index[item] = idx
	}

	return index
}

// GoodMakeMapWithCap uses make with capacity.
func GoodMakeMapWithCap(items []string) map[string]int {
	index := make(map[string]int, len(items))
	for idx, item := range items {
		index[item] = idx
	}

	return index
}

// GoodMapNotPopulatedInLoop is populated outside a loop.
func GoodMapNotPopulatedInLoop() map[string]int {
	index := map[string]int{}
	index["a"] = 1
	index["b"] = 2

	return index
}

// GoodSmallLiteralSlice ranges over a small literal slice (< 8 elements).
func GoodSmallLiteralSlice() map[string]int {
	index := map[string]int{}
	for idx, item := range []string{"a", "b", "c"} {
		index[item] = idx
	}

	return index
}

// GoodSmallLiteralSliceVar ranges over a small literal assigned to a variable.
func GoodSmallLiteralSliceVar() map[string]int {
	items := []string{"x", "y"}
	index := map[string]int{}
	for idx, item := range items {
		index[item] = idx
	}

	return index
}

// GoodSmallVarDecl uses a var declaration (ValueSpec path in rangeSourceLiteralLen).
func GoodSmallVarDecl() map[string]int {
	var items = []string{"a", "b"}
	index := map[string]int{}
	for idx, item := range items {
		index[item] = idx
	}

	return index
}

// GoodBoundarySevenElements ranges over exactly 7 elements (just under threshold).
func GoodBoundarySevenElements() map[string]int {
	index := map[string]int{}
	for idx, item := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		index[item] = idx
	}

	return index
}

// BadLargeLiteralSlice ranges over a literal with >= 8 elements.
func BadLargeLiteralSlice() map[string]int {
	index := map[string]int{} // want `map index populated in range loop without capacity hint`
	for idx, item := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		index[item] = idx
	}

	return index
}
