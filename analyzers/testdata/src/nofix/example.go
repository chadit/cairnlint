package nofix

// Box triggers the upstream modernize "any" analyzer, which is one of the
// analyzers that ships a suggested fix. The fixture exists to prove that fix
// never reaches a driver.
type Box struct {
	payload interface{} // want `interface\{\} can be replaced by any`
}

func Unwrap(box Box) interface{} { // want `interface\{\} can be replaced by any`
	return box.payload
}
