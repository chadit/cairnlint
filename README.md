# cairnlint

Custom Go static analysis tool built on
`golang.org/x/tools/go/analysis`. Standard analyzers
plus an agent-only tier for AI-assisted code review.
Covers scope-aware checks, loop-body rules, expression
patterns, and code quality enforcement. Generated files
(`.pb.go`, `/gen/`, `// Code generated ... DO NOT EDIT.`)
are automatically skipped.

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

# With an explicit build tag
cairnlint -tags=integration ./...

# Auto-discover every build tag in the tree and lint each configuration
cairnlint -tags=auto ./...

# Enable agent mode (heuristic analyzers for LLM triage)
cairnlint --agent ./...

# List every analyzer grouped by category (no analysis performed)
cairnlint --list
```

cairnlint resolves packages relative to the caller's
working directory. Run it from any Go module root.

### Build tags

Files behind `//go:build <tag>` are invisible to the
analysis framework unless the tag is set. Two flags
handle the usual cases.

`-tags=<value>` forwards the tag into `GOFLAGS` so
`go/packages` sees it when loading. Use this when you
know exactly which tag you want:

```bash
cairnlint -tags=integration ./test/integration/...
```

`-tags=auto` scans every `.go` file under the
target patterns, extracts user-defined build tags from
`//go:build` and `// +build` lines, then runs cairnlint
once per tag plus a default-build pass. Duplicate
diagnostics across passes collapse; exit code is the
worst observed. This is what you want in a full lint
script that should catch every issue regardless of tag:

```bash
cairnlint -tags=auto ./...
```

GOOS, GOARCH, compiler pseudo-tags (`cgo`, `race`,
`msan`, …), and `go1.N` version gates are filtered out
of auto-discovery since they are not user tags.
`testdata/`, `vendor/`, `node_modules/`, and hidden
directories are skipped during the scan.

## Agent Mode

cairnlint has a second tier of analyzers designed for
AI-assisted code review. These produce heuristic-based
diagnostics that have a higher false-positive rate than
the standard set. An LLM can read the output, check
usage patterns, and decide what's a real issue. A human
would find the noise distracting.

When agent mode is active, agent diagnostics are written
to a temp file at `/tmp/cairnlint-agent-<PID>.txt`
instead of stdout. A single summary line is printed to
stderr:

```text
[agent] heuristic findings written to /tmp/cairnlint-agent-12345.txt
```

Standard diagnostics still go to stdout as usual. If the
temp file can't be created, agent diagnostics fall back
to stdout.

### Enabling agent mode

There are three ways to enable it. Use whichever fits
your setup.

#### Pass the --agent flag

```bash
cairnlint --agent ./...
```

Works with any tool or script. The flag is stripped
before the analysis framework parses its own flags.

#### Auto-detection via environment variables

cairnlint checks for environment variables set by
popular AI coding tools. If any are present, agent
mode turns on automatically with no flag needed.

| Tool | Env var |
| ---- | ---- |
| Emerging standards | `AI_AGENT`, `AGENT` |
| Claude Code | `CLAUDECODE` |
| Codex CLI | `CODEX_SANDBOX`, `CODEX_THREAD_ID` |
| Gemini CLI | `GEMINI_CLI` |
| Cursor | `CURSOR_AGENT` |
| Qwen Code | `QWEN_CODE` |
| Goose | `GOOSE_TERMINAL` |
| Cline | `CLINE_ACTIVE` |
| Augment Code | `AUGMENT_AGENT` |
| TRAE AI | `TRAE_AI_SHELL_ID` |
| OpenCode | `OPENCODE_CLIENT` |
| Any tool | `CAIRNLINT_AGENT` |

#### Set CAIRNLINT_AGENT=1 for tools without auto-detection

Some tools (Aider, Continue.dev, Windsurf) don't set
identifying environment variables. For these, set
`CAIRNLINT_AGENT=1` in the tool's configuration:

```bash
# In your shell profile, tool config, or CI env
export CAIRNLINT_AGENT=1
```

### Setting env vars in specific tools

**Claude Code** already sets `CLAUDECODE=1` by default,
so auto-detection works out of the box. No setup needed.

**Codex CLI** sets `CODEX_SANDBOX` and `CODEX_THREAD_ID`
for every subprocess. Auto-detection works automatically.

**Gemini CLI** sets `GEMINI_CLI=1` in shell commands.
Auto-detection works automatically.

**Cursor** sets `CURSOR_AGENT=1`. Auto-detection works
automatically.

**Aider** does not set any identifying env vars. Add
this to your `.env` file or shell profile:

```bash
export CAIRNLINT_AGENT=1
```

**Continue.dev** does not set identifying env vars. Set
the variable in your Continue config or shell:

```bash
export CAIRNLINT_AGENT=1
```

**CI/CD pipelines** can set `CAIRNLINT_AGENT=1` when
cairnlint runs as part of an AI-powered review step, or
pass `--agent` directly.

### How the LLM uses agent output

The LLM's workflow looks like this:

1. Run `cairnlint ./...` (agent mode activates via env
   or flag)
2. See the stderr summary: `[agent] heuristic findings
   written to /tmp/cairnlint-agent-<PID>.txt`
3. Read the file to see full diagnostics
4. For each finding, check whether the flagged symbol has
   legitimate non-test consumers (e.g. `rg "SymbolName"
   --type go --glob '!*_test.go'`)
5. Report genuine issues, dismiss false positives

### Agent-only analyzers (2)

| Analyzer | What it flags |
| ---- | ---- |
| `agentexportedintestfile` | Exported decls in augmented `_test.go` files |
| `aibuzzwords` | AI-flavored vocabulary, hedging, and clichés in comments |

`agentexportedintestfile` flags exported func, var,
const, and type declarations in same-package test files
(package `foo`, not `foo_test`). Skips framework
functions: TestXxx, BenchmarkXxx, FuzzXxx, ExampleXxx,
TestMain.

`aibuzzwords` scans comments for vocabulary that the
writing-style rules forbid: buzzwords (`delve`, `robust`,
`leverage`), hedging (`generally speaking`, `it is worth
noting`), formal transitions (`furthermore`, `moreover`),
clichés (`in today's world`, `at its core`), and preachy
universals (`we all`, `everyone knows`). Hit rate is high
in technical prose, so this is agent-only; an LLM can
dismiss the legitimate uses of words like `critical` or
`extensive` without bothering a human reviewer.

### Adding a new agent-only analyzer

1. Create `analyzers/agent_<name>.go` with a constructor
   function
2. Create `analyzers/agent_<name>_test.go` using
   `findAgentAnalyzer` and `analysistest.Run`
3. Create `analyzers/testdata/src/<name>/` with fixture
   files containing `// want` comments
4. Register in `analyzers/agentmode.go` `AgentOnly()`
5. Run `go test ./analyzers/` and `go build .`

## Suppressing Diagnostics

Add a `//nolint` comment on the same line as the
diagnostic to suppress it. The name after `//nolint:`
is the **analyzer name** (e.g., `queryinloop`), not
the tool name. Analyzer names are listed in the tables
in the [Analyzers](#analyzers) section below.

```go
// Suppress a specific analyzer by name.
s := "" //nolint:prefervarzero

// Suppress multiple analyzers on the same line.
s := "" //nolint:prefervarzero,noelse

// Suppress all cairnlint analyzers on this line.
s := "" //nolint
```

Common examples:

```go
// Database cleanup that isn't an N+1 pattern.
db.Exec("TRUNCATE TABLE users") //nolint:queryinloop

// Intentional use of context.Background in production code.
db.QueryContext(context.Background(), query) //nolint:dbquerywithbarebackground

// Sentinel error in a file that isn't errors.go.
var ErrNotFound = errors.New("not found") //nolint:sentinelerrors
```

This works in both standalone mode and when running
as a golangci-lint plugin.

**Scope note:** a leading `//nolint` directive
suppresses every line of the node it attaches to. Because
the AST can attach one comment to more than one node
(e.g., a function and its body), suppression occasionally
covers a line or two beyond the most obvious target. The
behavior is always a superset of what a reader expects,
never a subset — if you need tighter scope, put the
directive on the specific line instead.

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

All analyzers run as part of the golangci-lint
pipeline alongside your other linters.

## Testing

```bash
make test          # go test -race -count=1 ./... (matches CI)
make test-fast     # same tests without -race for quick local loops
make help          # list every target
```

Tests use `analysistest.Run` with fixture files in
`analyzers/testdata/src/`. Each fixture contains
`// want` comments marking expected diagnostics.

Run from the module root. The `testdata/` directories
hold fixture code that is loaded by `analysistest.Run`,
not executed; running them directly (e.g. through an
IDE test explorer) can produce spurious failures since
they exist purely as syntactic input for the analyzers.

## Analyzers

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
| `nodefaulthttpclient` | `http.DefaultClient` usage (no timeout) |
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
| `reflectnokindcheck` | `reflect.Type.Fields()`/`NumField()` without Kind |
| `bufferpeekstore` | `Peek()` result used after buffer mutation (SSA) |
| `reflectinloop` | `reflect.ValueOf`/`TypeOf` inside loops |
| `benchreportallocs` | Benchmark missing `b.ReportAllocs()` |
| `benchresettimer` | Benchmark setup without `b.ResetTimer()` |
| `buildergrow` | `strings.Builder` in loop without `Grow()` |
| `mapprealloc` | Map populated in loop without capacity hint |
| `typednilerror` | Typed nil returned as error interface (SSA) |
| `stmtnoclose` | `db.Prepare` without `defer stmt.Close()` |
| `chandirection` | Bidirectional `chan T` in function params |

### Concurrency

| Analyzer | What it flags |
| ---- | ---- |
| `wgaddbeforego` | `wg.Add` before `wg.Go` (double-counts WaitGroup) |
| `gowggo` | `go wg.Go(...)` wrapping (races Add with Wait) |
| `wgdoneinwggo` | `wg.Done()` inside `wg.Go()` closure (double-decrement) |
| `preferwggo` | Pre-1.25 `wg.Add(1)` + `go func(){defer wg.Done()}()` pattern |
| `tickerleak` | `NewTicker`/`NewTimer` without `defer Stop()` |
| `chandirclose` | `close()` on bidirectional channel param |
| `poolresetbeforeput` | `sync.Pool.Put` without Reset (SSA) |

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
| `testcryptoinprod` | Test crypto packages in production code |
| `signalhandling` | `main()` with server but no signal handling |

### Documentation style

| Analyzer | What it flags |
| ---- | ---- |
| `emdash` | Em dash (U+2014) in comments |
| `docparamblock` | Javadoc `Parameters:`/`Returns:` blocks in doc comments |
| `doctutorialvoice` | Tutorial voice (`Lets you`, `Use this to`, `Here we`) |
| `teststructuredblock` | `Workflow:`/`Purpose:`/etc. in test doc comments |

## Adding a new analyzer

1. Create `analyzers/<name>.go` with a constructor
   function (not an exported var)
2. Create `analyzers/<name>_test.go` using
   `findAnalyzer` and `analysistest.Run`
3. Create `analyzers/testdata/src/<name>/` with fixture
   files containing `// want` comments
4. Register in `analyzers/analyzers.go` `All()`
5. Run `go test ./analyzers/` and `go build .`
