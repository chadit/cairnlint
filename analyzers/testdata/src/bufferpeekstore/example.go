package bufferpeekstore

import (
	"bytes"
	"fmt"
)

// BadPeekThenWrite stores Peek result, mutates buffer, then uses the stale slice.
func BadPeekThenWrite() {
	buf := bytes.NewBufferString("hello world")
	data, _ := buf.Peek(5) // want `bytes\.Buffer\.Peek result used after buffer mutation; the returned slice aliases internal memory and is now invalid`
	buf.WriteString(" extra")
	fmt.Println(data)
}

// BadPeekThenReset stores Peek result, resets buffer, then uses the stale slice.
func BadPeekThenReset() {
	buf := bytes.NewBufferString("hello")
	data, _ := buf.Peek(3) // want `bytes\.Buffer\.Peek result used after buffer mutation; the returned slice aliases internal memory and is now invalid`
	buf.Reset()
	fmt.Println(data)
}

// BadPeekThenRead stores Peek result, reads from buffer, then uses the stale slice.
func BadPeekThenRead() {
	buf := bytes.NewBufferString("hello world")
	data, _ := buf.Peek(5) // want `bytes\.Buffer\.Peek result used after buffer mutation; the returned slice aliases internal memory and is now invalid`
	buf.ReadByte() // #nosec G104 -- test fixture, error intentionally ignored
	fmt.Println(data)
}

// GoodPeekNoMutation uses Peek result without ever mutating the buffer.
func GoodPeekNoMutation() {
	buf := bytes.NewBufferString("hello world")
	data, _ := buf.Peek(5)
	fmt.Println(data)
}

// GoodPeekUsedBeforeMutation uses the result before any mutation happens.
func GoodPeekUsedBeforeMutation() {
	buf := bytes.NewBufferString("hello world")
	data, _ := buf.Peek(5)
	fmt.Println(data)
	buf.WriteString(" extra")
}

// GoodDifferentBuffers mutates a different buffer than the one Peek was called on.
func GoodDifferentBuffers() {
	buf1 := bytes.NewBufferString("hello")
	buf2 := bytes.NewBufferString("world")
	data, _ := buf1.Peek(3)
	buf2.WriteString(" extra")
	fmt.Println(data)
}
