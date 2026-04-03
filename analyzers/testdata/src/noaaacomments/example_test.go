package noaaacomments_test

import "testing"

func TestWithAAAComments(t *testing.T) {
	/* want `AAA section marker not allowed` */ // Arrange
	x := 1
	/* want `AAA section marker not allowed` */ // Act
	x++
	/* want `AAA section marker not allowed` */ // Assert
	if x != 2 {
		t.Fatal("unexpected")
	}
	_ = x
}

func TestWithNormalComments(t *testing.T) {
	// Set up the input value.
	x := 1

	// Arrange the furniture nicely -- not a standalone AAA marker.
	x++

	if x != 2 {
		t.Fatal("unexpected")
	}
}
