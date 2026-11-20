package agentloneparam

import "fmt"

// open receives three different paths that share their first segment, so the
// parameter varies.
func open(path string) string {
	return "open:" + path
}

// scale receives 0.5 at every call site.
func scale(value float64, factor float64) float64 { // want `^\[agent\] parameter factor of scale receives 0\.5 at all 2 call sites`
	return value * factor
}

// send receives the same content type, slash included, at every call site.
func send(body string, contentType string) string { // want `^\[agent\] parameter contentType of send receives "application/json" at all 2 call sites`
	return contentType + ":" + body
}

// RunConstants exercises the fixtures above.
func RunConstants() {
	fmt.Println(open("etc/hosts"), open("etc/passwd"), open("etc/group"))
	fmt.Println(scale(1, 0.5), scale(2, 0.5))
	fmt.Println(send("a", "application/json"), send("b", "application/json"))
}
