package urlclone

import (
	"maps"
	"net/url"
)

// Flagged: the copied map shares every []string with the original.
func copyValues(src url.Values) url.Values {
	return maps.Clone(src) // want `maps\.Clone on url\.Values copies the map but shares every \[\]string value`
}

// Flagged: the struct copy shares the *Userinfo pointer.
func copyURL(src *url.URL) url.URL {
	return *src // want `dereferencing a \*url\.URL copies the struct but shares the \*Userinfo`
}

// Flagged: same shallow copy through a local variable.
func mutateCopy(src *url.URL) *url.URL {
	dup := *src // want `dereferencing a \*url\.URL copies the struct but shares the \*Userinfo`
	dup.Path = "/replaced"

	return &dup
}

// Not flagged: maps.Clone on a map that holds no reference values.
func copyHeaders(src map[string]string) map[string]string {
	return maps.Clone(src)
}

// Not flagged: a pointer type in a declaration is not a dereference.
func passthrough(src *url.URL) *url.URL {
	var out *url.URL = src

	return out
}

// Not flagged: reading a field through the pointer copies only that field.
func readPath(src *url.URL) string {
	return src.Path
}
