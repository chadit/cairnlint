package literalsize

import (
	"os"
	"testing"
)

type buf []byte

// make is shadowed in localMake's scope only, to check a user-defined make.
func localMake() {
	make := func(n int) []int { return nil }

	_ = make(4)
}

// The make([]byte, 4096) line in os.WriteFile is reconstructed from dotai issue 793.
func TestSizes(t *testing.T) {
	devicePath := t.TempDir() + "/device"
	if err := os.WriteFile(devicePath, make([]byte, 4096), 0o644); err != nil { // want `literal size in make length; name it as a constant`
		t.Fatal(err)
	}

	ints := make([]int, 0, 16)        // want `literal size in make length; name it as a constant` `literal size in make capacity; name it as a constant`
	counts := make(map[string]int, 8) // want `literal size in make length; name it as a constant`
	events := make(chan int, 2)       // want `literal size in channel capacity; name it as a constant`
	named := make(buf, 4)             // want `literal size in make length; name it as a constant`
	plain := make([]byte, pageSize)
	unsized := make(map[string]int)

	_, _, _, _, _, _ = ints, counts, events, named, plain, unsized
}

func genericSlice[T ~[]int]() T {
	return make(T, 4) // want `literal size in make length; name it as a constant`
}

// A make inside a table row is still a call, so it reports in a test file.
func TestTableRow(t *testing.T) {
	cases := []struct {
		name string
		buf  []byte
	}{
		{name: "page", buf: make([]byte, 4096)}, // want `literal size in make length; name it as a constant`
	}

	for _, tc := range cases {
		t.Log(tc.name, len(tc.buf))
	}
}
