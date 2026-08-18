# API reference

Every exported symbol in `gitlab.com/phpboyscout/go/errorhandling`: what it does, what
it defaults to, and what happens when it is given something it did not expect.

[Report levels](levels.md), [exit codes](exit-codes.md) and the
[log fields](log-fields.md) a report emits have their own pages.

## Constructing a handler

### New

```go
func New(l *slog.Logger, help HelpConfig) ErrorHandler
```

Builds a [`StandardErrorHandler`](#standarderrorhandler) and returns it as an
[`ErrorHandler`](#errorhandler).

- **`l` must not be nil.** `New` does not check, and does not panic; the panic arrives
  later, at the first report, as a nil-pointer dereference inside `slog`. If the logger
  might be absent, pass `slog.New(slog.DiscardHandler)` rather than `nil`.
- **`help` may be nil**, and nil is the normal choice for a tool with no support
  channel. It disables the [`help` field](log-fields.md#help) entirely.

There is no exit function to inject. Nothing in this module terminates the process, so
a fatal path is testable without a seam — assert on what [`Fatal`](#fatal) returns.

### StandardErrorHandler

```go
type StandardErrorHandler struct {
	Logger *slog.Logger
	Help   HelpConfig
	Usage  func() error
}
```

The only implementation of [`ErrorHandler`](#errorhandler). Exported so it can be
constructed directly, but the zero value is **not** usable: a nil `Logger` panics on the
first report. Prefer [`New`](#new).

The fields are read live on each report, not copied at construction. Mutating any of
them while a report is in flight is a data race.

### ErrorHandler

```go
type ErrorHandler interface {
	Fatal(ctx context.Context, err error, opts ...ReportOption) int
	Error(ctx context.Context, err error, opts ...ReportOption)
	Warn(ctx context.Context, err error, opts ...ReportOption)
	SetUsage(usage func() error)
}
```

The reporting boundary. Take this interface in your own code rather than the concrete
type, so tests can substitute
[the published mock](../how-to/test-error-handling.md#mock-the-handler).

`Fatal` returns the exit code it believes the process should use. `Error` and `Warn`
return nothing — they are not terminal, so there is no code to report.

## Reporting an error

Every report produces **exactly one** `slog` record. A nil error produces none.

### Fatal

```go
func (h *StandardErrorHandler) Fatal(ctx context.Context, err error, opts ...ReportOption) int
```

Reports at `ERROR` and returns the [exit code](exit-codes.md). **It does not exit** —
`main` owns termination, because a library calling `os.Exit` skips every deferred
cleanup between itself and `main`.

`Fatal(ctx, nil)` returns `0` and logs nothing, so a guarded fatal path needs no second
nil check. A cancelled `ctx` does not suppress the report; the context is passed to
`slog` and used to ask whether debug is enabled.

### Error

```go
func (h *StandardErrorHandler) Error(ctx context.Context, err error, opts ...ReportOption)
```

Reports at `ERROR` and returns. The caller keeps control and the process keeps running.
Right for a failure worth telling someone about that does not end the run — one file in
a batch, one retry that did not work.

### Warn

```go
func (h *StandardErrorHandler) Warn(ctx context.Context, err error, opts ...ReportOption)
```

Reports at `WARN` and returns. For conditions that are not failures but are worth
surfacing: a deprecated flag, a fallback that had to be taken.

### SetUsage

```go
func (h *StandardErrorHandler) SetUsage(usage func() error)
```

Registers the function that prints usage when a reported error carries an
[`Outcome`](#outcome) with `Usage: true` — which is how
[`ErrRunSubCommand`](#errrunsubcommand) and
[`ErrUnknownSubCommand`](#errunknownsubcommand) reach a printer. Keeping this a plain
`func() error` is what lets the module stay free of any CLI-framework dependency; with
Cobra, pass `cmd.Usage`.

- The handler stores **one** function. The last call wins, which is why it belongs in a
  per-command pre-run hook rather than being set once at the root — see
  [Print usage for a parent command](../how-to/print-usage.md#set-it-per-command-not-once-globally).
- **The error the usage function returns is discarded.** A printer that fails does so
  silently; the report is still logged.
- Leaving it unset is valid. The error is still reported and the code is still the
  outcome's; there is simply no usage output.

## Report options

A `ReportOption` adjusts a single report. They are ordinary functions, applied in the
order given.

```go
type ReportOption func(*reportConfig)
```

### WithPrefix

```go
func WithPrefix(prefix string) ReportOption
```

Labels the report with the [`prefix` field](log-fields.md#prefix), for distinguishing
which phase of a run failed. It is attached as an attribute, **not** prepended to the
message. An empty prefix omits the field.

### Prefix

```go
func Prefix(parts ...string) string
```

Joins prefix fragments for a caller assembling one from parts. A convenience for
`WithPrefix`, not an option itself.

### Quietly

```go
func Quietly() ReportOption
```

Demotes the log line to `DEBUG` without changing the exit code, for an expected,
user-initiated end — an interrupt, say — where the non-zero code is the signal and an
error line would be noise.

**Read on the `Fatal` path only.** `Error(ctx, err, Quietly())` still logs at `ERROR`
and nothing reports that the option did nothing. An [`Outcome`](#outcome) also overrides
it, so a quiet report of `ErrRunSubCommand` still logs at `WARN`.

### WithStackDepth

```go
func WithStackDepth(frames int) ReportOption
```

Bounds how many frames of the [`stacktrace` field](log-fields.md#stacktrace) are
rendered. The frames kept are the innermost — the failure and its immediate callers.

A negative value removes the bound. This bounds **reporting**, not capture: `go/errors`
captures more, and `errors.StackOf(err)` still reaches all of it.

### DefaultStackDepth

```go
const DefaultStackDepth = 20
```

The bound applied when no `WithStackDepth` is given. Deep enough to cross several
packages and still show where a failure started, and deliberately smaller than what
`go/errors` captures.

## Outcomes

### Outcome

```go
type Outcome struct {
	Code    int
	Level   slog.Level
	Message string
	Usage   bool
}
```

How a terminal error should be presented and what exit code it deserves, declared beside
the error rather than switched on by the handler.

- **`Code`** is the process exit code. Zero is legitimate and meaningful: an outcome can
  be terminal *and* successful — a completed self-update must stop the run, and nothing
  went wrong.
- **`Level`** is how loudly to report it. A user-initiated stop is not an error and an
  interrupt is not a failure, so neither should log like one.
- **`Message`** replaces the error's own text, for when that text is machinery rather
  than something a user should read. Empty keeps the error's own.
- **`Usage`** prints the command's usage through the [`SetUsage`](#setusage) seam before
  reporting. It is a bool rather than a callback deliberately: an outcome carrying
  arbitrary behaviour would make an error a place to hide control flow.

### WithOutcome

```go
func WithOutcome(err error, o Outcome) error
```

Attaches an outcome to `err` without altering its message or identity. Returns `nil`
when `err` is nil. Transparent to `errors.Is`/`As`, and it survives further wrapping —
which is what lets a caller wrap one of this module's sentinels with context and keep
its ending.

### OutcomeOf

```go
func OutcomeOf(err error) (Outcome, bool)
```

Returns the **outermost** outcome in `err`'s tree, so a caller wrapping someone else's
error to change how it ends wins over the original. Reports `false` for an error with no
outcome, which leaves the level and code to the caller.

### OutcomeKind

```go
const OutcomeKind = "errorhandling.outcome"
```

Identifies an attached outcome to anything reading the wrapper directly.

It does **not** appear in [`errors.KindOf`](https://errors.go.phpboyscout.uk/reference/readers/#kindof).
The wrapper declares itself structural — an annotation rather than an identity —
so `KindOf` looks past it:

```go
errors.KindOf(ErrRunSubCommand)                       // "errorhandling.run_subcommand"
errors.KindOf(WithExitCode(ErrNotFound, 3))           // the sentinel's own kind
```

Before v0.5.0 it reported `errorhandling.outcome` for all three sentinels this
module ships — the plumbing, on exactly the errors most likely to be queried.

## Exit codes

### WithExitCode

```go
func WithExitCode(err error, code int) error
```

Attaches a process exit code to `err` without altering its message or identity. Returns
`nil` for a nil `err` — attaching a code to "no error" cannot invent a failure.

The wrapper is transparent to `errors.Is` and `errors.As`, and survives further
wrapping. `code` is not validated: any `int` is accepted, including values outside the
0–255 range a POSIX process can return, in which case the shell sees the low 8 bits.

An [`Outcome`](#outcome) on the same error beats it.

### ExitCode

```go
func ExitCode(err error) int
```

Reads the attached code back.

| Argument | Result |
|---|---|
| `nil` | `0` |
| An error with no attached code | `1` |
| An error with a code attached | that code |
| A code attached more than once | the **outermost**, matching `errors.As` order |

It traverses joined errors as well as wraps. It knows nothing about outcomes, so it and
[`Fatal`](#fatal) can legitimately disagree — see
[Exit codes](exit-codes.md#reading-a-code-back-in-a-test).

### ExitCodeUsage

```go
const ExitCodeUsage = 2
```

The conventional Unix "command misuse" status, carried by this module's three usage
sentinels. See [Exit codes](exit-codes.md#exitcodeusage).

### ExitCodeKind

```go
const ExitCodeKind = "errorhandling.exit_code"
```

Identifies an attached exit code to anything reading the wrapper directly. Like
[`OutcomeKind`](#outcomekind) it is structural, so it does not mask the error's
identity in `errors.KindOf`.

## Sentinels

### ErrNotImplemented

```go
var ErrNotImplemented = WithOutcome(
	errors.NewSentinel("errorhandling.not_implemented", "command not yet implemented"),
	Outcome{Code: ExitCodeUsage, Level: slog.LevelWarn},
)
```

A command that exists but does nothing yet. Reported at `WARN` whatever method reports
it, because a stub is not a failure, and [`Fatal`](#fatal) returns
[`2`](#exitcodeusage). No usage is printed.

### NewErrNotImplemented

```go
func NewErrNotImplemented(issueURL string) error
```

`ErrNotImplemented` with a stack captured at the call site, plus `issueURL` as an
`issue_url` attribute — which arrives inside the [`err` group](log-fields.md#err) of the
same record, not on a line of its own.

- **The result *is* `errors.Is(err, ErrNotImplemented)`**, so code matching on the
  sentinel matches errors from this constructor.
- **`issueURL` is not validated.** An empty string adds no attribute at all; a non-URL
  string is carried as given.

### ErrRunSubCommand

```go
var ErrRunSubCommand = WithOutcome(
	errors.NewSentinel("errorhandling.run_subcommand", "subcommand required"),
	Outcome{Code: ExitCodeUsage, Level: slog.LevelWarn, Usage: true},
)
```

A parent command invoked with no subcommand, **where that is itself the mistake**. The
handler calls the [`SetUsage`](#setusage) printer, logs at warn, and `Fatal` returns
[`ExitCodeUsage`](#exitcodeusage).

Returning it is a choice, not a rule. A parent that only groups its children can equally
treat a bare invocation as a request for help and succeed; reach for this when the
command genuinely cannot act on its own.

Detection is `errors.Is`, and a wrap's own message **is** reported —
`errors.Wrap(ErrRunSubCommand, "config")` logs `config: subcommand required`. The
outcome travels through the wrap with it.

### ErrUnknownSubCommand

```go
var ErrUnknownSubCommand = WithOutcome(
	errors.NewSentinel("errorhandling.unknown_subcommand", "unknown subcommand"),
	Outcome{Code: ExitCodeUsage, Level: slog.LevelWarn, Usage: true},
)
```

A parent command given a verb it does not have. Distinct from
[`ErrRunSubCommand`](#errrunsubcommand): there, no subcommand was given at all; here,
one was, and it does not exist.

Cobra reports an unknown command for the **root only**, so a parent that wants to catch
a mistyped subcommand has to do it in its own run function. Wrap the sentinel with the
offending verb and the command path:

```go
errors.Wrapf(errorhandling.ErrUnknownSubCommand,
	"unknown command %q for %q", args[0], cmd.CommandPath())
```

which reports `unknown command "bogus" for "tool alpha": unknown subcommand`, prints
usage, and returns `2`.

### UnknownSubCommand

```go
func UnknownSubCommand(verb, path string) error
```

Wraps [`ErrUnknownSubCommand`](#errunknownsubcommand) with the offending verb and
the command that rejected it:

```go
errorhandling.UnknownSubCommand("bogus", "tool alpha")
// unknown command "bogus" for "tool alpha": unknown subcommand
```

It exists so the wording and the sentinel do not drift between CLIs. The cobra
glue that calls it deliberately lives elsewhere — this module imports no CLI
framework, and a module holding eight lines of glue would not earn its keep — so
what is shared is the message and the identity, not the closure:

```go
RunE: func(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return errorhandling.UnknownSubCommand(args[0], cmd.CommandPath())
	}

	return cmd.Usage()
}
```

### ErrAssertionFailure

```go
var ErrAssertionFailure = errors.NewSentinel(
	"errorhandling.assertion_failure", "internal invariant violated")
```

A violated internal invariant — a bug in the program rather than a mistake by its user.

It carries **no** outcome, so it does not override the reporting level and `Fatal`
returns `1` unless a code is attached. What identifies it is the kind on the record,
which a query can filter on and a string prefix cannot.

### NewAssertionFailure

```go
func NewAssertionFailure(format string, args ...any) error
```

Builds an error wrapping [`ErrAssertionFailure`](#errassertionfailure) with the
formatted message. Reported like any other error: one record, at the requested level.

## Supplying a support channel

### HelpConfig

```go
type HelpConfig interface {
	SupportMessage() string
}
```

The extension point for a support-channel message. The module ships the interface and no
implementations, deliberately — see
[Add a support channel](../how-to/support-channel.md#why-the-module-ships-no-implementations).

- `SupportMessage` is called **on every report**, so it must be cheap. Resolve the
  message once at startup and return the stored value.
- Returning `""` suppresses the [`help` field](log-fields.md#help) — that is the
  designed way to stay silent when a channel is not configured.
- Passing `nil` for the whole interface disables help output without any nil handling on
  your side.
- It is called on the reporting goroutine with no context and no cancellation. It must
  not block, and it must be safe to call concurrently.

## Log field key constants

`KeyError`, `KeyPrefix`, `KeyHelp` and `KeyStacktrace` hold the field names the handler
uses. They are exported so tests can assert on a key without hard-coding the string. See
[Log fields](log-fields.md) for what each field contains and when it is present.

Hints, details, attributes and the error's kind have no key constants here: they travel
inside the [`err` group](log-fields.md#err) as the error's own `slog.LogValuer` output,
not as fields this module adds.
