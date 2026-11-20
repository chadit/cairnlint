package singlecallwrapper

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	_ "unsafe"
)

var errProgressAborted = errors.New("progress aborted")

type settings struct{ count int }

var config = settings{count: 2}

var compareFn = strings.Compare

// Reported: each parameter reaches the call once, in order, beside fixed arguments.

func callbackAborted(err error) error { // want `^function callbackAborted only returns fmt\.Errorf called with its own parameters and fixed arguments; call fmt\.Errorf directly at each caller$`
	return fmt.Errorf("%w: %w", errProgressAborted, err)
}

func newRequest(url string) (*http.Request, error) { // want `function newRequest only returns http\.NewRequest`
	return http.NewRequest(http.MethodGet, url, nil)
}

func errorf(format string, args ...any) error { // want `function errorf only returns fmt\.Errorf`
	return fmt.Errorf(format, args...)
}

func copyOut(r io.Reader) (int64, error) { // want `function copyOut only returns io\.Copy`
	return io.Copy(os.Stdout, r)
}

func upper(s string) string { // want `function upper only returns strings\.Map`
	return strings.Map(unicode.ToUpper, s)
}

func hex(n int64) string { // want `function hex only returns strconv\.FormatInt`
	return strconv.FormatInt(n, 16)
}

func both(a, b string) int { // want `function both only returns strings\.Compare`
	return strings.Compare(a, b)
}

// Not reported: the wrapper's own shape.

func Exported(err error) error {
	return fmt.Errorf("wrap: %w", err)
}

type store struct{}

func (store) wrap(err error) error {
	return fmt.Errorf("store: %w", err)
}

func contains[T comparable](s []T, v T) bool {
	return slices.Contains(s, v)
}

func guarded(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("guarded: %w", err)
}

func show(s string) {
	fmt.Println(s)
}

func exit(code int) {
	os.Exit(code)
}

func shout(s string) string {
	return strings.ToUpper(s) + "!"
}

func now() time.Time {
	return time.Now()
}

func reader(s string) io.Reader {
	return strings.NewReader(s)
}

func open(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

//go:linkname linked
func linked(s string) string {
	return strings.ToLower(s)
}

// The directive sits apart from pulled so that only its missing body keeps
// it unreported.
//go:linkname pulled strings.ToUpper

func pulled(s string) string

func bare(s string) (out string) {
	return
}

func parse(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

var parser func(string) (int64, error) = parse

// Not reported: the callee is not a function another package declares.

func local(s string) string {
	return helper(s)
}

func helper(s string) string {
	return strings.TrimSpace(s) + "."
}

func render(b *strings.Builder) string {
	return b.String()
}

func write(s string) (int, error) {
	return os.Stdout.WriteString(s)
}

func size(s string) int {
	return len(s)
}

func viaVar(a, b string) int {
	return compareFn(a, b)
}

// Not reported: a parameter does not reach the call once, unchanged, in order.

func dropped(s string, n int) string {
	return strings.Repeat(s, 2)
}

func blank(s string, _ int) string {
	return strings.ToUpper(s)
}

func transformed(n int) string {
	return strconv.Itoa(n + 1)
}

func twice(s string) int {
	return strings.Compare(s, s)
}

func reversed(a, b string) int {
	return strings.Compare(b, a)
}

func join(parts ...string) string {
	return strings.Join(parts, ",")
}

// Not reported: an extra argument is not a constant, nil, or package-level name.

func trimmed(s string) (cut string) {
	return strings.TrimPrefix(s, cut)
}

func repeated(s string) string {
	return strings.Repeat(s, config.count)
}

func stamped(s string) string {
	return fmt.Sprint(s, time.Now())
}

// Run calls every fixture so none is unused.
func Run() {
	_ = callbackAborted(nil)
	_, _ = newRequest("/")
	_ = errorf("x")
	_, _ = copyOut(nil)
	_ = upper("x")
	_ = hex(1)
	_ = both("a", "b")
	_ = store{}.wrap(nil)
	_ = contains([]int{1}, 1)
	_ = guarded(nil)
	show("x")
	_ = shout("x")
	defer exit(0)
	_ = now()
	_ = reader("x")
	_, _ = open("x")
	_ = linked("x")
	_ = pulled("x")
	_ = bare("x")
	_, _ = openFile("x")
	_, _ = createFile("x")
	_ = itoa(1)
	_ = parser
	_ = local("x")
	_ = render(nil)
	_ = size("x")
	_, _ = write("x")
	_ = viaVar("a", "b")
	_ = dropped("x", 1)
	_ = blank("x", 1)
	_ = transformed(1)
	_ = twice("x")
	_ = reversed("a", "b")
	_ = join("a")
	_ = trimmed("x")
	_ = repeated("x")
	_ = stamped("x")
}
