package testunderscoreprefix_test

import "testing"

// Flagged: underscore immediately after the Test prefix (empty subject).
func Test_Something(t *testing.T) { // want `test name "Test_Something" has an underscore immediately after the Test prefix`
	t.Log("bad name")
}

// Flagged: still empty subject even with a trailing scenario segment.
func Test_Something_one(t *testing.T) { // want `test name "Test_Something_one" has an underscore immediately after the Test prefix`
	t.Log("bad name")
}

// Not flagged: subject "Load" precedes the underscore.
func TestLoad_MissingGrpcBind(t *testing.T) {
	t.Log("good name")
}

// Not flagged: subject "Example" precedes the underscore.
func TestExample_something(t *testing.T) {
	t.Log("good name")
}

// Not flagged: MixedCaps with no underscore.
func TestLoadConfig(t *testing.T) {
	t.Log("good name")
}

// Not flagged: bare Test with no suffix.
func Test(t *testing.T) {
	t.Log("bare Test")
}

// Not flagged: Benchmark grouping is left alone.
func Benchmark_Something(b *testing.B) {
	b.Log("benchmark")
}

// Not flagged: Fuzz grouping is left alone.
func Fuzz_Something(f *testing.F) {
	f.Log("fuzz")
}

// Not flagged: helper that isn't a Test function.
func helper_function() {}
