package chandirection

// BadBidirectional has bidirectional channel params.
func BadBidirectional(ch chan int) {} // want `bidirectional chan in parameter`

// BadMultiParam has multiple params, one bidirectional.
func BadMultiParam(done chan struct{}, items []int) {} // want `bidirectional chan in parameter`

// GoodReceiveOnly uses directional channel.
func GoodReceiveOnly(ch <-chan int) {}

// GoodSendOnly uses directional channel.
func GoodSendOnly(ch chan<- int) {}

// GoodNoChannel has no channel params.
func GoodNoChannel(x int) {}

// GoodReturnChannel returns a channel (factory pattern, acceptable).
func GoodReturnChannel() chan int { return make(chan int) }
