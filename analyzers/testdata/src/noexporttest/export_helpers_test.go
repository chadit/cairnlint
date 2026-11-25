package noexporttest_test // a test file whose name only starts like export_test.go

import "testing"

func TestHelpers(t *testing.T) {
	t.Log("a test file with another name is allowed")
}
