package nounderscoretest_test

import "testing"

// Flagged: underscores in test name.
func TestFoo_Bar(t *testing.T) { // want `test name "TestFoo_Bar" contains underscores; use MixedCaps`
	t.Log("bad name")
}

// Flagged: multiple underscores.
func TestFoo_Bar_Baz(t *testing.T) { // want `test name "TestFoo_Bar_Baz" contains underscores; use MixedCaps`
	t.Log("bad name")
}

// Not flagged: MixedCaps with no underscores.
func TestFooBar(t *testing.T) {
	t.Log("good name")
}

// Not flagged: just "Test" with no suffix is valid.
func Test(t *testing.T) {
	t.Log("bare Test")
}

// Flagged: benchmark with underscore.
func BenchmarkFoo_Bar(b *testing.B) { // want `test name "BenchmarkFoo_Bar" contains underscores; use MixedCaps`
	b.Log("bad name")
}

// Not flagged: benchmark without underscore.
func BenchmarkFooBar(b *testing.B) {
	b.Log("good name")
}

// Flagged: fuzz with underscore.
func FuzzParse_Input(f *testing.F) { // want `test name "FuzzParse_Input" contains underscores; use MixedCaps`
	f.Log("bad name")
}

// Not flagged: fuzz without underscore.
func FuzzParseInput(f *testing.F) {
	f.Log("good name")
}

// Not flagged: helper function with underscore (not a test function).
func helper_function() {}
