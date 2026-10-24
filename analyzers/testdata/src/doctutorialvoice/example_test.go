package doctutorialvoice

import "testing"

// Run inside the test container (root, partclone and tmpfs needed). go test // want `a sentence that opens with "Run"`
// only runs benchmarks under -bench, so make test never picks this up:
//
//	docker compose exec imagenext-test go test ./internal/partclone/ -run "^$" -bench BenchmarkRestoreStdin -benchtime=5x -benchmem // want `run instruction in test doc comment`
func BenchmarkRestoreStdin(b *testing.B) {
	for b.Loop() {
		Zero()
	}
}

// TestRestore checks a restore round trip.
//
//	go test ./... -run TestRestore // want `run instruction in test doc comment`
func TestRestore(t *testing.T) {
	Zero()
}

// runAll runs the whole suite the way CI does:
//
//	go test ./...
func runAll(t *testing.T) {
	t.Helper()
}

// TestRestoreShort checks the short restore path, run with
//	go test -short ./... // want `run instruction in test doc comment`
func TestRestoreShort(t *testing.T) {
	Zero()
}

// TestListed covers both cache states:
//   - an empty go test cache
//     and then a warm go test cache
func TestListed(t *testing.T) {
	Zero()
}

// TestSpaceTab checks a code line indented by a space and then a tab.
//
// 	go test -run TestSpaceTab ./... // want `run instruction in test doc comment`
func TestSpaceTab(t *testing.T) {
	Zero()
}

// Test is a valid test name, since cmd/go accepts a bare prefix.
//
//	go test -run '^Test$' ./... // want `run instruction in test doc comment`
func Test(t *testing.T) {
	Zero()
}

// Testdata is a helper whose lowercase suffix keeps it from being a test.
//
//	go test ./...
func Testdata() {}

// ExampleZero shows the empty buffer, checked by
//
//	go test -run ExampleZero
func ExampleZero() {
	Zero()
	// Output:
}

// TestCache covers:
//   - a cold cache
//
//     then go test reruns with a warm one
func TestCache(t *testing.T) {
	Zero()
}

// TestBullets covers:
//   • a go test run with a bullet marker
func TestBullets(t *testing.T) {
	Zero()
}

// TestAfterList covers:
//   - a cold cache
//
// The prose line above closes the list, so this line is code:
//
//	go test -run TestAfterList ./... // want `run instruction in test doc comment`
func TestAfterList(t *testing.T) {
	Zero()
}
