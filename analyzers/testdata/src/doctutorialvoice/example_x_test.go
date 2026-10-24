package doctutorialvoice_test

import (
	"reflect"

	"doctutorialvoice"
)

// Job carries a hook whose field name a doc sentence may use.
type Job struct {
	Run func()
}

// New returns a Job. Run defaults to a no-op.
func New() Job { return Job{Run: func() {}} }

// Callable promotes reflect.Value's methods, Call among them.
type Callable struct {
	reflect.Value
}

// Forward wraps a reflected function.
// Call forwards to the wrapped value.
func Forward(fn any) Callable { return Callable{reflect.ValueOf(fn)} }

// Wrap drives a driver from outside the package.
// Execute is the tested package's method.
func Wrap(driver doctutorialvoice.Driver) { driver.Execute() }
