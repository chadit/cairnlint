package preferany

// Flagged: empty interface as a struct field.
type Bad struct {
	Field interface{} // want `use any instead of interface\{\}`
}

// Flagged: empty interface as a parameter.
func TakesEmpty(x interface{}) {} // want `use any instead of interface\{\}`

// Not flagged: any reads as an identifier, not an interface literal.
func TakesAny(x any) {}

// Not flagged: a non-empty interface is a real interface.
type Stringer interface {
	String() string
}
