package chandirclose

// BadCloseBidirectionalParam closes a bidirectional channel param.
func BadCloseBidirectionalParam(ch chan int) {
	close(ch) // want `close\(\) on bidirectional channel parameter`
}

// GoodCloseSendOnly closes a send-direction channel (sender owns it).
func GoodCloseSendOnly(ch chan<- int) {
	close(ch)
}

// GoodCloseLocalChannel closes a locally created channel.
func GoodCloseLocalChannel() {
	ch := make(chan int)
	close(ch)
}

// GoodNoClose doesn't close anything.
func GoodNoClose(ch chan int) {
	ch <- 1
}
