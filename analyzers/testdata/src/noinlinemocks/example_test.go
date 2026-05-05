package noinlinemocks_test

import "testing"

// Bad: mock struct type inline in a test file.
type MockService struct{} // want `mock type MockService must be in a mocks/ directory`

func (MockService) DoWork() error { return nil }

// Bad: lowercase mock prefix.
type mockRepo struct{} // want `mock type mockRepo must be in a mocks/ directory`

// Good: not a mock (no Mock/mock prefix).
type fakeService struct{}

func TestPlaceholder(t *testing.T) {
	_ = MockService{}
	_ = mockRepo{}
	_ = fakeService{}
}
