package notestifysuites_test

import (
	"notestifysuites/suite"
	"testing"
)

// Flagged: embedding suite.Suite in a test struct.
type MySuite struct {
	suite.Suite // want `do not embed suite\.Suite`
}

// Not flagged: named field (not an embed).
type HasSuiteField struct {
	mySuite suite.Suite
}

// Not flagged: struct without suite embedding.
type PlainHelper struct {
	Name string
}

func TestPlaceholder(t *testing.T) {
	_ = MySuite{}
	_ = HasSuiteField{}
	_ = PlainHelper{}
}
