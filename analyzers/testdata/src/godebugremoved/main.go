//go:debug asynctimerchan=1
//go:debug tlsrsakex=1
//go:debug panicnil=1

// Both removed settings report on the package clause: a //go:debug value may
// not contain a space, so the directive lines cannot carry a want comment.
package main // want `GODEBUG setting asynctimerchan was removed in Go 1\.27` `GODEBUG setting tlsrsakex was removed in Go 1\.27`

func main() {}
