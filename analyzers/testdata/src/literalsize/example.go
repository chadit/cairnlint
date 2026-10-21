package literalsize

const bufSize = 32

// The [2]uint64 and make(chan error, 1) lines are reconstructed from dotai issue 793.
type header struct {
	magic [16]byte // want `literal size in array length; name it as a constant`
	_     [0]func()
	buf   [bufSize]byte
}

func discard(offset, length uint64) [2]uint64 { // want `literal size in array length; name it as a constant`
	discardRange := [2]uint64{offset, length} // want `literal size in array length; name it as a constant`

	return discardRange
}

func produce() chan error {
	producerErr := make(chan error, 1) // want `literal size in channel capacity; name it as a constant`

	return producerErr
}

func local() {
	var scratch [4]byte // want `literal size in array length; name it as a constant`

	_ = scratch
}

// mnd owns call-argument sizes outside tests.
func outsideTests() ([]byte, map[string]int) {
	return make([]byte, 4096), make(map[string]int, 8)
}

func notSizes(n int) {
	primes := [...]int{1, 2}
	sized := [bufSize + 1]int{}
	dynamic := make([]byte, n)
	unbuffered := make(chan int)

	_, _, _, _ = primes, sized, dynamic, unbuffered
}

// A function literal's body is ordinary code even inside a file-scope var.
var handler = func() {
	ch := make(chan int, 4) // want `literal size in channel capacity; name it as a constant`

	_ = ch
}

func genericChan[C ~chan int]() C {
	return make(C, 2) // want `literal size in channel capacity; name it as a constant`
}

// An embedded comparable restricts no kind, so the chan term still decides.
func genericComparable[C interface {
	comparable
	~chan int
}]() C {
	return make(C, 3) // want `literal size in channel capacity; name it as a constant`
}
