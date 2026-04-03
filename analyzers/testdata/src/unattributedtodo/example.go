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

// Not flagged: TODO embedded in an identifier reference.
// The context.TODO() function returns a non-nil empty context.

// Not flagged: TODO/FIXME in prose, not as a task directive.
// This analyzer flags TODO/FIXME/HACK/XXX markers at comment start.

// Placeholder keeps the package compilable.
func Placeholder() {}
