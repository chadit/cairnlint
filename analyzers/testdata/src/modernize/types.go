package modernize

import (
	"reflect"
	"sync/atomic"
	"unsafe"
)

// Box holds a value of any type, spelled the pre-1.18 way.
type Box struct {
	Value interface{}
}

// Stringer is a non-empty interface, which any cannot replace.
type Stringer interface {
	String() string
}

// TypeOfInt asks reflect for a type through a zero value.
func TypeOfInt() reflect.Type {
	return reflect.TypeOf(0)
}

// TypeOfValue asks reflect for the dynamic type of a value it was handed,
// which TypeFor cannot know, so reflecttypefor stays quiet.
func TypeOfValue(value any) reflect.Type {
	return reflect.TypeOf(value)
}

// AsInt type-asserts through Interface, which reflect.TypeAssert replaces.
func AsInt(value reflect.Value) (int, bool) {
	number, ok := value.Interface().(int)

	return number, ok
}

// AsAny keeps the interface value, so reflecttypeassert stays quiet.
func AsAny(value reflect.Value) any {
	return value.Interface()
}

// Inner is a struct that omitempty never omits.
type Inner struct {
	Count int
}

// Outer tags a struct field omitempty, which omitzero replaces.
type Outer struct {
	Inner Inner `json:"inner,omitempty"`
}

// Label tags a string omitempty, which does omit the empty string, so
// omitzero stays quiet.
type Label struct {
	Name string `json:"name,omitempty"`
}

// Base is embedded in Derived.
type Base struct {
	ID int
}

// Derived embeds Base.
type Derived struct {
	Base

	Label string
}

// NewDerived names the embedded field in the literal, which embedlit drops.
func NewDerived() Derived {
	return Derived{Base: Base{ID: 1}, Label: "first"}
}

// NewLabel sets only the outer field, so embedlit stays quiet.
func NewLabel() Derived {
	return Derived{Label: "second"}
}

// Counter is bumped through the sync/atomic functions that atomic.Int64
// replaces.
type Counter struct {
	hits int64
}

// Hit increments the counter.
func (c *Counter) Hit() {
	atomic.AddInt64(&c.hits, 1)
}

// Gauge is read and written plainly, so atomictypes stays quiet.
type Gauge struct {
	level int64
}

// Set stores a level.
func (g *Gauge) Set(level int64) {
	g.level = level
}

// Next steps a pointer with uintptr arithmetic that unsafe.Add replaces.
func Next(ptr unsafe.Pointer) unsafe.Pointer {
	return unsafe.Pointer(uintptr(ptr) + 1)
}

// Address converts without arithmetic, so unsafefuncs stays quiet.
func Address(ptr unsafe.Pointer) uintptr {
	return uintptr(ptr)
}
