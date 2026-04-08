package poolresetbeforeput

import (
	"bytes"
	"sync"
)

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// BadPutWithoutReset returns dirty buffer to pool.
func BadPutWithoutReset() {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.WriteString("data")
	bufPool.Put(buf) // want `sync\.Pool\.Put without Reset`
}

// GoodPutWithReset resets before returning.
func GoodPutWithReset() {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.WriteString("data")
	buf.Reset()
	bufPool.Put(buf)
}

// GoodPutNewObject puts a freshly created object.
func GoodPutNewObject() {
	bufPool.Put(new(bytes.Buffer))
}
