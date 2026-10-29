// Fixture for TestAgentFileAbsentWithoutFindings. contextbackground reports
// the call below, and the //nolint drops it before the agent file sees it, so
// the wrapper runs but nothing reaches the writer.
package agentoutputclean_test

import (
	"context"
	"testing"
)

func TestSuppressedByNolint(t *testing.T) {
	_ = context.Background() //nolint:contextbackground // must not create the agent file
	_ = t
}

func TestUsesTestContext(t *testing.T) {
	_ = t.Context()
}
