package synctestsleepwait_test

import (
	"testing"
	"testing/synctest"
	"time"
)

// Flagged: Sleep then Wait is exactly what synctest.Sleep does.
func TestSleepThenWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		go func() {}()

		time.Sleep(time.Second) // want `time\.Sleep followed by synctest\.Wait can be replaced with synctest\.Sleep`
		synctest.Wait()
	})
}

// Not flagged: a statement between the two runs before the bubble settles, so
// the single call would reorder it.
func TestStatementBetween(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		count := 0

		time.Sleep(time.Second)

		count++

		synctest.Wait()

		_ = count
	})
}

// Not flagged: Sleep with no following Wait.
func TestSleepAlone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(time.Second)
	})
}

// Not flagged: Wait on its own.
func TestWaitAlone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		go func() {}()
		synctest.Wait()
	})
}
