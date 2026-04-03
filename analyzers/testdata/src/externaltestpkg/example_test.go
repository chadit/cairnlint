package externaltestpkg // want `test file must use external test package`

import "testing"

func TestSomething(t *testing.T) {
	t.Log("this test uses an internal package")
}
