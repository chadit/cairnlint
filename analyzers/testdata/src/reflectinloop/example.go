package reflectinloop

import "reflect"

// BadValueOfInLoop calls reflect.ValueOf inside a for loop.
func BadValueOfInLoop(items []any) {
	for _, item := range items {
		_ = reflect.ValueOf(item) // want `reflect\.ValueOf\(\) inside loop`
	}
}

// BadTypeOfInLoop calls reflect.TypeOf inside a for loop.
func BadTypeOfInLoop(items []any) {
	for _, item := range items {
		_ = reflect.TypeOf(item) // want `reflect\.TypeOf\(\) inside loop`
	}
}

// GoodValueOfOutsideLoop is fine, not in a loop.
func GoodValueOfOutsideLoop(item any) {
	_ = reflect.ValueOf(item)
}

// GoodClosureInsideLoop wraps in a closure, different scope.
func GoodClosureInsideLoop(items []any) {
	for _, item := range items {
		func() {
			_ = reflect.ValueOf(item)
		}()
	}
}
