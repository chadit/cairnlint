# cairnlint

Custom Go static analysis tool built on
`golang.org/x/tools/go/analysis`. 31 analyzers covering
scope-aware checks, loop-body rules, expression patterns,
and code quality enforcement.

## Install

```bash
go install github.com/chadit/cairnlint@latest
```

Or from a local checkout:

```bash
git clone git@github.com:chadit/cairnlint.git
cd cairnlint
go install .
```

Both put the `cairnlint` binary in `$GOBIN`
(usually `~/go/bin/`).

## Usage

```bash
# Analyze all packages in the current module
cairnlint ./...

# Single package
cairnlint ./internal/config/...

# Disable a specific analyzer
cairnlint -noelse=false ./...

# Enable only one analyzer
cairnlint -synctestsleep ./...

# With build tags
cairnlint -tags=integration ./...
```

cairnlint resolves packages relative to the caller's
working directory. Run it from any Go module root.

## golangci-lint Integration

cairnlint can run as a module plugin inside a custom
golangci-lint build.

### 1. Create `.custom-gcl.yml`

```yaml
version: v2.11.4
plugins:
  - module: 'github.com/chadit/cairnlint'
    import: 'github.com/chadit/cairnlint/plugin'
```

For local development, use a path reference:

```yaml
version: v2.11.4
plugins:
  - module: 'github.com/chadit/cairnlint'
    path: '/path/to/cairnlint'
```

### 2. Build custom golangci-lint

```bash
golangci-lint custom
```

This produces a `custom-gcl` binary with cairnlint
built in.

### 3. Configure `.golangci.yml`

```yaml
version: "2"

linters:
  enable:
    - cairnlint
  settings:
    custom:
      cairnlint:
        type: "module"
        description: "Custom Go analysis rules"
```

### 4. Run

```bash
./custom-gcl run ./...
```

All 31 analyzers run as part of the golangci-lint
pipeline alongside your other linters.

## Testing

```bash
cd cmd/cairnlint
go test ./analyzers/ -v
```

Tests use `analysistest.Run` with fixture files in
`analyzers/testdata/src/`. Each fixture contains
`// want` comments marking expected diagnostics.

## Analyzers (31)

### Scope-dependent (synctest exemption)

| Analyzer | What it flags |
| ---- | ---- |
| `synctestsleep` | `time.Sleep` in tests outside synctest |
| `contextbackground` | `context.Background()` in tests |
| `contexttodo` | `context.TODO()` in tests |
| `wrappedcontextbackground` | `context.With*(ctx.Background())` |

### Loop-body and structural

| Analyzer | What it flags |
| ---- | ---- |
| `deferinloop` | `defer` inside `for` loops |
| `queryinloop` | `db.Query/Exec` and variants in loops |
| `stringconcatinloop` | `s += x` or `s = s + x` in loops |
| `preferbloop` | `b.N` loops in benchmarks |
| `dbquerywithbarebackground` | `db.*Context(ctx.Background())` |
| `noelse` | `if-else` blocks (use early returns) |

### Expression-level

| Analyzer | What it flags |
| ---- | ---- |
| `nounderscoretest` | `TestFoo_Bar` naming |
| `noruntimenumgoroutine` | `runtime.NumGoroutine()` in tests |
| `nogenericerror` | `errors.New("error")` vague messages |
| `noerrstrcontains` | `Contains(err.Error(), ...)` |
| `nopanicinlib` | `panic()` in non-test files |
| `nocontextinstruct` | `context.Context` as struct field |
| `prefererrorsastype` | `errors.As` (use `errors.AsType`) |
| `preferfmtappendf` | `[]byte(fmt.Sprintf(...))` |
| `typeassertnocheck` | `x := y.(Type)` without comma-ok |
| `notestifysuites` | `suite.Suite` embedding |
| `prefervarzero` | `s := ""` (use `var s string`) |

### Code quality

| Analyzer | What it flags |
| ---- | ---- |
| `commentedcode` | Disabled Go code in comments |
| `discardedcontext` | `_ context.Context` in params |
| `sentinelerrors` | Sentinel errors outside `errors.go` |
| `sqlinjection` | `fmt.Sprintf` with SQL keywords |
| `externaltestpkg` | Internal test package names |
| `noexporttest` | `export_test.go` files |
| `nofortestfunc` | `ForTest`/`ForTesting` suffixes |
| `noaaacomments` | `// Arrange/Act/Assert` comments |
| `noinlinemocks` | Inline `MockFoo` structs in tests |
| `unattributedtodo` | Unowned TODO/FIXME/HACK/XXX |

## Adding a new analyzer

1. Create `analyzers/<name>.go` with a constructor
   function (not an exported var)
2. Create `analyzers/<name>_test.go` using
   `findAnalyzer` and `analysistest.Run`
3. Create `analyzers/testdata/src/<name>/` with fixture
   files containing `// want` comments
4. Register in `analyzers/analyzers.go` `All()`
5. Run `go test ./analyzers/` and `go build .`
