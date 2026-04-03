package unattributedtodo

// No owner or ticket reference.
// TODO add error handling // want `task marker without owner or ticket`

// FIXME this is broken // want `task marker without owner or ticket`

// Has owner in parentheses.
// TODO(alice) add error handling

// Has ticket reference.
// TODO PROJ-123 add error handling

// Has owner.
// FIXME(bob) this is broken

// Not a task marker at all.
// This is a regular comment.

// Placeholder keeps the package compilable.
func Placeholder() {}
