package analyzers

import "errors"

// errNotRegularFile reports that the agent file path holds something other
// than a regular file, such as a FIFO, so the writer refused it.
var errNotRegularFile = errors.New("not a regular file")
