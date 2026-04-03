package synctestsleep_test

import (
	"testing"
	"testing/synctest"
	"time"
)

// Flagged: time.Sleep outside synctest.
func TestSleepOutsideSynctest(t *testing.T) {
	time.Sleep(time.Second) // want `time\.Sleep in tests is a flaky test signal`
}

// Not flagged: time.Sleep inside synctest.Test closure.
func TestSleepInsideSynctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(5 * time.Minute)
	})
}

// Mixed: one flagged, one not.
func TestMixed(t *testing.T) {
	time.Sleep(time.Second) // want `time\.Sleep in tests is a flaky test signal`
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(5 * time.Minute)
	})
}

// Flagged: time.Sleep in a plain closure (not synctest).
func TestSleepInPlainClosure(t *testing.T) {
	f := func() {
		time.Sleep(time.Second) // want `time\.Sleep in tests is a flaky test signal`
	}
	f()
}

// Not flagged: nested closure inside synctest.Test.
func TestNestedClosureInSynctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		go func() {
			time.Sleep(time.Second)
		}()
	})
}
