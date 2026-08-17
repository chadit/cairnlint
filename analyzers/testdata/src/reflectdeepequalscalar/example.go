package reflectdeepequalscalar

import (
	"reflect"
	"time"
	"unsafe"
)

// Color is a named scalar type; == works on it, so DeepEqual is still redundant.
type Color int

// Pair is a composite type; DeepEqual is the right tool for it.
type Pair struct {
	Left, Right int
}

// BadIdenticalScalars compares same-typed scalars with DeepEqual.
func BadIdenticalScalars(a, b int, s, t string, x, y bool, f, g float64, c, d Color, dur, limit time.Duration) {
	_ = reflect.DeepEqual(a, b)         // want `reflect\.DeepEqual on int operands; use == or != for scalar types`
	_ = reflect.DeepEqual(s, t)         // want `reflect\.DeepEqual on string operands`
	_ = reflect.DeepEqual(x, y)         // want `reflect\.DeepEqual on bool operands`
	_ = reflect.DeepEqual(f, g)         // want `reflect\.DeepEqual on float64 operands`
	_ = reflect.DeepEqual(c, d)         // want `reflect\.DeepEqual on reflectdeepequalscalar\.Color operands`
	_ = reflect.DeepEqual(dur, limit)   // want `reflect\.DeepEqual on time\.Duration operands`
	_ = reflect.DeepEqual(a, 5)         // want `reflect\.DeepEqual on int operands`
	_ = reflect.DeepEqual(s, "literal") // want `reflect\.DeepEqual on string operands`

	if !reflect.DeepEqual(a, b) { // want `reflect\.DeepEqual on int operands`
		return
	}
}

// BadMismatchedTypes compares distinct concrete types, which DeepEqual never
// equates, so every call is dead code that == would have refused to compile.
func BadMismatchedTypes(a int, wide int64, c Color, s string, f float64, p Pair) {
	_ = reflect.DeepEqual(a, wide) // want `reflect\.DeepEqual on int and int64 is always false`
	_ = reflect.DeepEqual(c, a)    // want `reflect\.DeepEqual on reflectdeepequalscalar\.Color and int is always false`
	_ = reflect.DeepEqual(wide, 5) // want `reflect\.DeepEqual on int64 and int is always false`
	_ = reflect.DeepEqual(f, 1)    // want `reflect\.DeepEqual on float64 and int is always false`
	_ = reflect.DeepEqual(p, s)    // want `reflect\.DeepEqual on reflectdeepequalscalar\.Pair and string is always false`
	_ = reflect.DeepEqual(s, nil)  // want `reflect\.DeepEqual on string and untyped nil is always false`
}

// GoodComposites uses DeepEqual on values == cannot compare or compares
// structurally; none of these should be flagged.
func GoodComposites(p, q Pair, xs, ys []int, m, n map[string]int, raw, cooked []byte, ptr, other *int, u, v unsafe.Pointer) {
	_ = reflect.DeepEqual(p, q)
	_ = reflect.DeepEqual(xs, ys)
	_ = reflect.DeepEqual(m, n)
	_ = reflect.DeepEqual(raw, cooked)
	_ = reflect.DeepEqual(ptr, other)
	_ = reflect.DeepEqual(u, v)
}

// GoodInterfaceOperands has at least one operand whose static type is an
// interface, so the runtime type is unknown and DeepEqual is a fair choice.
func GoodInterfaceOperands(got, want any, obj map[string]any, err error, code Color) {
	_ = reflect.DeepEqual(got, want)
	_ = reflect.DeepEqual(obj["kind"], "Status")
	_ = reflect.DeepEqual(err, code)
	_ = reflect.DeepEqual(recover(), 0xdead)
}

// GoodTypeParam compares type parameters; T may not be comparable, so == is
// not guaranteed to compile.
func GoodTypeParam[T any](a, b T) bool {
	return reflect.DeepEqual(a, b)
}

// GoodTypeParamScalar mixes a type parameter with a scalar. T is not
// identical to int even when instantiated with it, so this must not be
// reported as always false; only the interface guard keeps it quiet.
func GoodTypeParamScalar[T ~int](a T, b int) bool {
	return reflect.DeepEqual(a, b)
}

// DeepEqual is a local function that shadows the reflect name; the analyzer
// resolves through type info and must not match it.
func DeepEqual(a, b int) bool {
	return a == b
}

// GoodLocalDeepEqual calls the local DeepEqual, not reflect's.
func GoodLocalDeepEqual(a, b int) bool {
	return DeepEqual(a, b)
}
