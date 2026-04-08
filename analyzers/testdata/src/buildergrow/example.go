package buildergrow

import "strings"

// BadBuilderInLoop writes in loop without Grow.
func BadBuilderInLoop(items []string) string {
	var sb strings.Builder
	for _, item := range items {
		sb.WriteString(item) // want `strings\.Builder\.WriteString\(\) in loop without Grow`
	}
	return sb.String()
}

// BadWriteByteInLoop uses WriteByte in loop without Grow.
func BadWriteByteInLoop(data []byte) string {
	var sb strings.Builder
	for _, b := range data {
		sb.WriteByte(b) // want `strings\.Builder\.WriteByte\(\) in loop without Grow`
	}
	return sb.String()
}

// BadWriteRuneInLoop uses WriteRune in loop without Grow.
func BadWriteRuneInLoop(runes []rune) string {
	var sb strings.Builder
	for _, r := range runes {
		sb.WriteRune(r) // want `strings\.Builder\.WriteRune\(\) in loop without Grow`
	}
	return sb.String()
}

// BadWriteInLoop uses Write in loop without Grow.
func BadWriteInLoop(chunks [][]byte) string {
	var sb strings.Builder
	for _, chunk := range chunks {
		sb.Write(chunk) // want `strings\.Builder\.Write\(\) in loop without Grow`
	}
	return sb.String()
}

// GoodBuilderWithGrow has Grow before loop.
func GoodBuilderWithGrow(items []string) string {
	var sb strings.Builder
	sb.Grow(len(items) * 10)
	for _, item := range items {
		sb.WriteString(item)
	}
	return sb.String()
}

// GoodBuilderOutsideLoop uses Builder outside a loop.
func GoodBuilderOutsideLoop() string {
	var sb strings.Builder
	sb.WriteString("hello")
	return sb.String()
}

// GoodBuilderInClosureInsideLoop wraps the body in a closure so the builder
// belongs to the closure's scope, not the outer function.
func GoodBuilderInClosureInsideLoop(items []string) []string {
	var results []string
	for _, item := range items {
		func() {
			var sb strings.Builder
			sb.WriteString(item)
			results = append(results, sb.String())
		}()
	}
	return results
}
