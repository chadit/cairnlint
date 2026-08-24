//go:debug asynctimerchan=0
//go:debug tlsrsakex=0
//go:debug panicnil=1

// Removed settings sit at their post-removal default because the go command
// refuses to load a package pinning any other value. Reports land on the
// package clause since a //go:debug value may not contain a space.
package main // want `GODEBUG setting asynctimerchan was removed in Go 1\.27` `GODEBUG setting tlsrsakex was removed in Go 1\.27`

func main() {}
