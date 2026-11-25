//go:build !unix

package analyzers

// agentOpenFlags adds nothing where O_NOFOLLOW and O_NONBLOCK do not exist,
// so the tool still builds there. The regular-file check after the open still
// refuses a non-file entry, but it cannot stop a symlink: on Windows the open
// follows the link and truncates its target before the check runs.
const agentOpenFlags = 0
