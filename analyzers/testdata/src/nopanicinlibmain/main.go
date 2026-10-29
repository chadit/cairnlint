package main

import "os"

// Not flagged: init in package main may fail fast at startup.
func init() {
	if os.Getenv("REQUIRED") == "" {
		panic("REQUIRED is unset")
	}
}

// Not flagged: main has no caller to return an error to.
func main() {
	if len(os.Args) > 3 {
		panic("too many arguments")
	}

	done := make(chan struct{})

	// Flagged: the goroutine body runs as library code.
	go func() {
		defer close(done)

		panic("worker failed") // want `panic\(\) in library code; return an error instead`
	}()

	<-done

	run()
}

// Flagged: an ordinary function in package main returns errors like any other.
func run() {
	panic("helper failed") // want `panic\(\) in library code; return an error instead`
}
