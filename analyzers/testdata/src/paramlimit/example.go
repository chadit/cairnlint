package paramlimit

import (
	"context"
	stdctx "context"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Reported: the interface method from dotai issue 909 has 4 counted parameters.
type Downloader interface {
	Download(ctx context.Context, targetImageObjectID, endpointURL, localPath string, datacenterID int32) error // want `Download has 4 domain parameters, more than the limit of 3; group them in a request or config struct, or use functional options`
}

// Reported: a struct field written as a func type literal counts the same way.
type Client struct {
	DownloadFunc func(ctx context.Context, targetImageObjectID, endpointURL, localPath string, datacenterID int32) error // want `DownloadFunc has 4 domain parameters`
}

// Reported: the diagnostic lands on the name of a signature split one group per line.
func download( // want `download has 4 domain parameters`
	ctx context.Context,
	targetImageObjectID, endpointURL, localPath string,
	datacenterID int32,
) error {
	return nil
}

// Reported: only the first logger is exempt, so the second makes 4.
func twoLoggers(ctx context.Context, log *slog.Logger, audit *slog.Logger, name, email string, age int) error { // want `twoLoggers has 4 domain parameters`
	return nil
}

// Reported: a variadic of data counts one.
func dataVariadic(name, email string, age int, ids ...string) error { // want `dataVariadic has 4 domain parameters`
	return nil
}

type Store struct{}

// Reported: the receiver is not a parameter, and the 4 others count.
func (s *Store) Save(ctx context.Context, name, email string, age int, role string) error { // want `Save has 4 domain parameters`
	return nil
}

// Reported: two loggers in one group exempt only the first.
func groupedLoggers(log, audit *slog.Logger, name, email string, age int) error { // want `groupedLoggers has 4 domain parameters`
	return nil
}

// Reported: unnamed parameters count one each.
type Sink interface {
	Put(string, string, int, bool) error // want `Put has 4 domain parameters`
}

// Reported: a defined type over context.Context is not context.Context, so
// it counts.
type MyCtx context.Context

func definedContext(ctx MyCtx, name, email string, age int) error { // want `definedContext has 4 domain parameters`
	return nil
}

// Not reported: the ACCEPTABLE examples from dotai lang-rules/golang/10_code_organization.md.
type CreateUserOpts struct {
	Name, Email string
	Age         int
	Role        string
	Active      bool
}

func CreateUser(opts CreateUserOpts) error {
	return nil
}

type ProxyDefaults struct{}

type CreateProxyRequest struct{}

func parseCreateProxyRequest(ctx context.Context, log *slog.Logger, r *http.Request, accountID string, defaults ProxyDefaults) (*CreateProxyRequest, error) {
	return nil, nil
}

type Options struct{}

type Option func(*Options)

func WithForce(cleanUp func(), exit func(error), force func() error, timeout time.Duration) Option {
	return func(o *Options) {}
}

func serialDownload(ctx context.Context, client *http.Client, url string, dest io.Writer, isRetryable func(error) bool) error {
	return nil
}

// Not reported: exactly 3 counted parameters.
func exactlyThree(name, email string, age int) error {
	return nil
}

// Not reported: a named func type is a behavior seam.
type CleanupFunc func()

func namedFuncType(name, email string, age int, cleanup CleanupFunc) error {
	return nil
}

// Not reported: context through an aliased import and through a type alias.
type Ctx = context.Context

func aliasedImport(ctx stdctx.Context, name, email string, age int) error {
	return nil
}

func typeAlias(ctx Ctx, name, email string, age int) error {
	return nil
}

// Not reported: functional options are a variadic of a func type.
type Conn struct{}

func New(addr, user, pass string, opts ...Option) *Conn {
	return nil
}

// Not reported: a project's own Logger interface is the exempt logger.
type Logger interface {
	Info(msg string)
}

func ownLogger(log Logger, name, email string, age int) error {
	return nil
}

// Not reported: a pointer to a non-slog type whose name ends in Logger.
type AuditLogger struct{}

func auditLogger(log *AuditLogger, name, email string, age int) error {
	return nil
}

// Not reported: a struct field of a named func type is out of scope, as are
// func literals and `type X func(...)` declarations.
type DownloadFunc func(ctx context.Context, targetImageObjectID, endpointURL, localPath string, datacenterID int32) error

type Hooks struct {
	Download DownloadFunc
}

var literal = func(name, email string, age int, role string) error {
	return nil
}

// Not reported: an interface-typed option variadic, as gRPC's DialOption is.
type DialOption interface {
	apply(*Options)
}

func Dial(target, user, pass string, opts ...DialOption) *Conn {
	return nil
}

// Reported: an option type that is not variadic is a domain parameter.
func dialOne(target, user, pass string, opt DialOption) *Conn { // want `dialOne has 4 domain parameters`
	return nil
}

// Not reported: every context is exempt, not only the first.
func twoContexts(ctx context.Context, parent context.Context, name, email string, age int) error {
	return nil
}

// Not reported: an unnamed logger is still the exempt logger.
func reportUnnamed(*slog.Logger, string, string, string) {}

type Reporter interface {
	Report(*slog.Logger, string, string, string)
}

// Not reported: a logger reached through a type alias.
type Log = slog.Logger

func aliasLogger(log *Log, name, email string, age int) error {
	return nil
}

// Not reported: a variadic of a func literal type is a set of hooks.
func withHooks(name, email string, age int, hooks ...func()) error {
	return nil
}

// Not reported: an interface method and a func-typed field at exactly 3
// after exemptions.
type Fetcher interface {
	Fetch(ctx context.Context, log *slog.Logger, url, dest string, retries int) error
}

type Fetchers struct {
	FetchFunc func(ctx context.Context, url, dest string, retries int, cleanup func()) error
}

// Reported: each name in a field that declares several names is checked.
type MultiHooks struct {
	Before, After func(name, email string, age int, role string) error // want `Before has 4 domain parameters` `After has 4 domain parameters`
}
