//go:build ignore

package main

import "strconv"

func quote(s string) string {
	return strconv.Quote(s) + "\n"
}

func main() { println(quote("x")) }
