package externaltestpkg_test // the external test package the analyzer asks for

import "testing"

func TestExternal(t *testing.T) {
	t.Log("this test uses the external test package")
}
