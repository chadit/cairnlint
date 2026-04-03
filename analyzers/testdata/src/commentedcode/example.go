package commentedcode

import "fmt"

// SomeFunc does something useful.
func SomeFunc() {
	fmt.Println("hello")
}

// func disabledFunc() {} // want `commented-out code`

// var old = "unused" // want `commented-out code`

// return early // want `commented-out code`

// This is a normal comment, not code.
func AnotherFunc() {
	// when we decide to add logging later, revisit this
	fmt.Println("world")
}

// defer cleanup() // want `commented-out code`

// SafeFunc is documented with a doc comment.
// if the input is valid we proceed.
func SafeFunc() {}
