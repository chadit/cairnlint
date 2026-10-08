// Fixture for TestAgentFileHonorsFilters. No want comments: the wrapper
// under test routes every diagnostic to the agent file instead of the
// analysis driver, and the test reads that file.
package agentoutput_test

import (
	"context"
	"testing"
)

func TestSuppressedByNolint(t *testing.T) {
	_ = context.Background() //nolint:contextbackground // must not reach the agent file
	_ = t
}

func TestReportedToAgentFile(t *testing.T) {
	_ = context.Background()
	_ = t
}
