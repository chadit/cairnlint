//go:build unix

package analyzers

import "syscall"

// agentOpenFlags guard the predictable agent file path: O_NOFOLLOW refuses a
// symlink planted there, and O_NONBLOCK keeps a planted FIFO with no reader
// from blocking the open forever.
const agentOpenFlags = syscall.O_NOFOLLOW | syscall.O_NONBLOCK
