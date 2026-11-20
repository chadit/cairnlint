package agentoutputdup

import "testing"

func TestValidate(t *testing.T) {
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
}
