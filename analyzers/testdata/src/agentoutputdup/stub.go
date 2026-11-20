// Package agentoutputdup has a test file, so the driver analyzes stub.go once
// in the plain package and again in its test variant.
package agentoutputdup

// Validate checks the configuration.
func Validate() error {
	return nil
}
