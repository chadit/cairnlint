//go:build unix

package singlecallwrapper

import "os"

// createFile is a variable in create_windows.go, so callers cannot inline it.
func createFile(name string) (*os.File, error) {
	return os.Create(name)
}
