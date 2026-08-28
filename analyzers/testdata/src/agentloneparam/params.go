package agentloneparam

import "fmt"

// render is called three times and strict is true every time.
func render(tpl string, strict bool) string { // want `parameter strict of render receives true at all 3 call sites`
	if strict {
		return "strict:" + tpl
	}

	return tpl
}

// format varies its mode across call sites, so the parameter earns its place.
func format(tpl string, mode string) string {
	if mode == "wide" {
		return "wide:" + tpl
	}

	return tpl
}

// once is called a single time, which proves nothing about variation.
func once(tpl string, strict bool) string {
	if strict {
		return "strict:" + tpl
	}

	return tpl
}

// escaping is called with a fixed value but is also passed as a value, so the
// arguments at its other invocations are invisible.
func escaping(tpl string, strict bool) string {
	if strict {
		return "strict:" + tpl
	}

	return tpl
}

// Exported callees can be reached from other packages, so they are not counted.
func Exported(tpl string, strict bool) string {
	if strict {
		return "strict:" + tpl
	}

	return tpl
}

// Run exercises the fixtures above.
func Run() {
	fmt.Println(render("page", true), render("header", true), render("footer", true))
	fmt.Println(format("page", "wide"), format("header", "narrow"))
	fmt.Println(once("page", true))
	fmt.Println(escaping("page", true), escaping("header", true))
	fmt.Println(Exported("page", true), Exported("header", true))

	var fn func(string, bool) string = escaping

	fmt.Println(fn("footer", false))
}
