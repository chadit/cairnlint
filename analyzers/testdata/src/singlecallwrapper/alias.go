package singlecallwrapper

import sc "strconv"

func itoa(n int) string { // want `^function itoa only returns sc\.Itoa called with its own parameters and fixed arguments; call sc\.Itoa directly at each caller$`
	return sc.Itoa(n)
}
