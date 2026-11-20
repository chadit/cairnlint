//go:build unix

package singlecallwrapper

import "os"

// openFile is declared again in plat_windows.go around another callee, so
// callers cannot inline either one.
func openFile(name string) (*os.File, error) {
	return os.Open(name)
}
