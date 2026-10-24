package doctutorialvoice

// LetsYouDoIt Lets you trigger the workflow from a caller. // want `tutorial voice`
func LetsYouDoIt() {}

// UseThisFunc Use this to bundle the setup into one call. // want `tutorial voice`
func UseThisFunc() {}

// HereWeGo Here we configure the client before sending. // want `tutorial voice`
func HereWeGo() {}

// AllowsYouFunc Allows you to inject extra options at runtime. // want `tutorial voice`
func AllowsYouFunc() {}

// GoodFunc builds a configured client ready for upload and download
// operations. Callers choose whichever operation fits their workflow.
func GoodFunc() {}

// placeholder keeps the package compilable.
func placeholder() {}

// Driver is the type whose method doc sentences may name.
type Driver struct{}

// Execute is the method the RunAll doc names.
func (Driver) Execute() {}

// RunAll calls Execute on every driver.
//
// Execute is wrapped by RunAll.
func RunAll(drivers []Driver) {
	for _, driver := range drivers {
		driver.Execute()
	}
}

// OldClient builds a v1 client.
//
// Deprecated: the v1 client is frozen.
// Use NewClient instead.
func OldClient() {}

// Parse reads a request header.
// See RFC 7230, section 3.3.
func Parse() {}

// Unmarshal decodes a payload.
// See [Marshal] for the encoding rules.
func Unmarshal() {}

// Marshal encodes a payload.
func Marshal() {}

// Zero returns an empty buffer. Use of the zero value is safe.
func Zero() {}

// composeUp runs the equivalent of
//
//	docker compose up -d --wait
func composeUp() {}

// Container holds the test container settings.
// Run inside the container.
type Container struct{}

// Reader closes over a stream.
// Call Close when the stream is done. // want `a sentence that opens with "Call"`
func Reader() {}

// Stream waits for the job to
// Call back into the pool.
func Stream() {}

// Runner is the interface whose method a doc sentence may name.
type Runner interface{ Invoke() }

// Dispatch hands work to a runner.
// Invoke is called once per item.
func Dispatch(runner Runner) { runner.Invoke() }

// Lock guards the shared state. Call sites must hold mu.
func Lock() {}

// Batch groups writes. Use cases include batching.
func Batch() {}

// Scan walks the input once. Run time is O(n).
func Scan() {}

// Setup wires the fakes. Use it before every case. // want `a sentence that opens with "Use"`
func Setup() {}

// helperDocs keeps the unexported helpers referenced.
var helperDocs = []func(){composeUp, placeholder}

// TestShapedName lives outside a _test.go file, so its code block stays.
//
//	go test ./...
func TestShapedName() {}

// Shutdown stops the server in this order:
//
//  1. Run the drain hooks.
//  2. Call each closer.
func Shutdown() {}
