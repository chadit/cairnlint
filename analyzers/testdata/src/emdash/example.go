package emdash

// This comment contains an em dash — and should be flagged. // want `em dash`

// This one uses a regular hyphen - which is fine.

/* Block comment with em dash — is also flagged. */ // want `em dash`

// Placeholder keeps the package compilable.
func Placeholder() {}
