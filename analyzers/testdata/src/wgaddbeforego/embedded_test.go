package wgaddbeforego

import "sync"

type worker struct {
	sync.WaitGroup // embedded
}

func BadEmbeddedWaitGroup() {
	w := worker{}
	w.Add(1) // want `wg.Add before wg.Go is redundant; WaitGroup.Go calls Add\(1\) internally, this double-counts and hangs Wait`
	w.Go(func() {})
	w.Wait()
}
