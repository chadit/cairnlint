package stringconcatinloop

// BadPlusAssignInLoop builds a string with += inside a range loop.
func BadPlusAssignInLoop(items []string) string {
	var result string
	for _, item := range items {
		result += item // want `string concatenation with \+= inside loop; use strings\.Builder instead`
	}

	return result
}

// BadPlusAssignInCStyleLoop uses a C-style loop.
func BadPlusAssignInCStyleLoop(n int) string {
	var result string
	for idx := 0; idx < n; idx++ {
		result += "x" // want `string concatenation with \+= inside loop; use strings\.Builder instead`
	}

	return result
}

// BadExplicitConcatInLoop uses s = s + x form inside a loop.
func BadExplicitConcatInLoop(items []string) string {
	var result string
	for _, item := range items {
		result = result + item // want `string concatenation with \+= inside loop; use strings\.Builder instead`
	}

	return result
}

// GoodPlusAssignOutsideLoop is fine because it's not inside a loop.
func GoodPlusAssignOutsideLoop() string {
	result := "hello"
	result += " world"

	return result
}

// GoodIntPlusAssign is fine because the variable is an int, not a string.
func GoodIntPlusAssign(nums []int) int {
	var sum int
	for _, num := range nums {
		sum += num
	}

	return sum
}
