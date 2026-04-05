package agentexportedintestfile

import "testing"

// Exported var in augmented test file — should be flagged.
var ExportedHelper = placeholder // want `exported var/const ExportedHelper in augmented _test.go file`

// Exported const — should be flagged.
const ExportedConst = 42 // want `exported var/const ExportedConst in augmented _test.go file`

// Exported type — should be flagged.
type ExportedType struct{} // want `exported type ExportedType in augmented _test.go file`

// Exported non-test func — should be flagged.
func SetupDB() {} // want `exported func SetupDB in augmented _test.go file`

// unexported func — should NOT be flagged.
func helperSetup() {}

// unexported var — should NOT be flagged.
var internalHelper = placeholder

// TestSomething is a framework function — should NOT be flagged.
func TestSomething(t *testing.T) {
	t.Log("ok")
}

// BenchmarkSomething is a framework function — should NOT be flagged.
func BenchmarkSomething(b *testing.B) {
	b.Log("ok")
}

// ExamplePlaceholder is a framework function — should NOT be flagged.
func ExamplePlaceholder() {}

// TestMain is a framework entry point — should NOT be flagged.
func TestMain(m *testing.M) {
	m.Run()
}
