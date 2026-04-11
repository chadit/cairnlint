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

// GoodSmallLiteralRange writes in a range loop over a small literal.
func GoodSmallLiteralRange() string {
	var sb strings.Builder
	for _, item := range []string{"a", "b", "c"} {
		sb.WriteString(item)
	}
	return sb.String()
}

// GoodSmallLiteralVar writes in a range loop over a small literal variable.
func GoodSmallLiteralVar() string {
	items := []string{"x", "y"}
	var sb strings.Builder
	for _, item := range items {
		sb.WriteString(item)
	}
	return sb.String()
}

// GoodBoundarySevenLiteral ranges over exactly 7 elements (just under threshold).
func GoodBoundarySevenLiteral() string {
	var sb strings.Builder
	for _, item := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		sb.WriteString(item)
	}
	return sb.String()
}

// BadLargeLiteralRange ranges over a literal with >= 8 elements.
func BadLargeLiteralRange() string {
	var sb strings.Builder
	for _, item := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		sb.WriteString(item) // want `strings\.Builder\.WriteString\(\) in loop without Grow`
	}
	return sb.String()
}

// BadForStmtLoop uses a C-style for loop (always flagged, can't determine count).
func BadForStmtLoop(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString("x") // want `strings\.Builder\.WriteString\(\) in loop without Grow`
	}
	return sb.String()
}

// GoodBuilderDeclaredInsideLoop declares the builder inside the loop body.
// Each iteration creates a fresh builder with no cross-iteration accumulation.
func GoodBuilderDeclaredInsideLoop(items []string) []string {
	var results []string
	for _, item := range items {
		var sb strings.Builder
		sb.WriteString(item)
		sb.WriteString(" suffix")
		results = append(results, sb.String())
	}
	return results
}

// GoodBuilderShortVarInsideLoop uses := to declare the builder inside the loop.
func GoodBuilderShortVarInsideLoop(items []string) []string {
	var results []string
	for _, item := range items {
		sb := strings.Builder{}
		sb.WriteString(item)
		results = append(results, sb.String())
	}
	return results
}

// GoodBuilderInsideForStmt declares the builder inside a C-style for loop.
func GoodBuilderInsideForStmt(n int) []string {
	var results []string
	for i := 0; i < n; i++ {
		var sb strings.Builder
		sb.WriteString("item")
		results = append(results, sb.String())
	}
	return results
}
