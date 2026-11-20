package singlecallwrapper

import "syscall"

func openFile(name string) (syscall.Handle, error) {
	return syscall.Open(name, 0, 0)
}
