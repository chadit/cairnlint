package singlecallwrapper

import "strconv"

func quote(s string) string { // want `function quote only returns strconv\.Quote`
	return strconv.Quote(s)
}

var _ = quote("x")
