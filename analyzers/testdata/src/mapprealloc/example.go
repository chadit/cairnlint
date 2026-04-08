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
