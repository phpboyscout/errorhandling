# API reference

Every exported symbol in `gitlab.com/phpboyscout/go/errorhandling`: what it does, what
it defaults to, and what happens when it is given something it did not expect.

Levels have their own page ([Report levels](levels.md)), as do
[exit codes](exit-codes.md) and the [log fields](log-fields.md) a report emits.

## Constructing a handler

### New

```go
func New(l *slog.Logger, help HelpConfig, opts ...Option) ErrorHandler
```

Builds a `StandardErrorHandler` with `Exit` set to `os.Exit`, then applies `opts` in
order. Returns it as an `ErrorHandler`.

- **`l` must not be nil.** `New` does not check, and does not panic; the panic arrives
  later, at the first report, as a nil-pointer dereference inside `slog`. If the logger
  might be absent, pass `slog.New(slog.DiscardHandler)` rather than `nil`.
- **`help` may be nil**, and nil is the normal choice for a tool with no support
  channel. It disables the [`help` field](log-fields.md#help) entirely.
- **`opts` may be empty.** The only option shipped is [`WithExitFunc`](#withexitfunc).

### WithExitFunc

```go
func WithExitFunc(exit ExitFunc) Option
```

Replaces `os.Exit` as the way the handler terminates the process. This is the seam that
makes fatal paths testable — see
[Test error handling](../how-to/test-error-handling.md#never-let-fatal-kill-your-tests).

Passing `nil` is accepted and produces a handler that panics when it reaches a fatal
report, so do not use it to mean "never exit"; pass `func(int) {}` instead.

**A fake exit function returns, and real `os.Exit` does not.** Code after a `Fatal`
call keeps running under a fake, including code written on the assumption that the
process had already gone.

### ExitFunc

```go
type ExitFunc func(code int)
```

The process-termination seam. `os.Exit` satisfies it directly.

### Option

```go
type Option func(*StandardErrorHandler)
```

A functional option applied by [`New`](#new) after the defaults are set. Because both
the type and [`StandardErrorHandler`](#standarderrorhandler) are exported, you can write
your own option outside the package — `func(h *errorhandling.StandardErrorHandler) { … }`
satisfies it — and set any of the four fields that way. Only `WithExitFunc` ships here.

### StandardErrorHandler

```go
type StandardErrorHandler struct {
	Logger *slog.Logger
	Help   HelpConfig
	Exit   ExitFunc
	Usage  func() error
}
```

The only implementation of [`ErrorHandler`](#errorhandler). Exported so it can be
constructed directly — but the zero value is **not** usable: `Logger` nil panics on the
first report, and `Exit` nil panics after logging a fatal report, which is the worse of
the two because the failure is reported and *then* the process dies the wrong way.
Prefer [`New`](#new), which fills both.

The fields are read live on each report, not copied at construction: `Logger` always,
`Exit` on any fatal-level report, `Help` on the ordinary path, `Usage` on the
special-error path. Mutating any of them while a report is in flight is a data race.

## Reporting an error

### ErrorHandler

```go
type ErrorHandler interface {
	Check(err error, prefix string, level string)
	Fatal(err error, prefixes ...string)
	Error(err error, prefixes ...string)
	Warn(err error, prefixes ...string)
	SetUsage(usage func() error)
}
```

The reporting boundary. Take this interface in your own code rather than the concrete
type, so tests can substitute
[the published mock](../how-to/test-error-handling.md#mock-the-handler).

None of the reporting methods returns anything. A report either logs, or logs and
exits; there is no error path to check.

### Check

```go
func (h *StandardErrorHandler) Check(err error, prefix string, level string)
```

The single entry point every other reporting method funnels into.

- **`err == nil` is a no-op** — nothing is logged and nothing exits. `Check(doThing(),
  "", LevelError)` is safe without a preceding nil test.
- **`prefix`** is attached as the structured [`prefix` field](log-fields.md#prefix)
  when non-empty. It is *not* prepended to the message.
- **`level`** is one of the [level constants](levels.md). An unrecognised string logs
  at error and does not exit, rather than being dropped.

The sentinels that carry an [`Outcome`](#outcome) — [`ErrRunSubCommand`](#errrunsubcommand),
[`ErrUnknownSubCommand`](#errunknownsubcommand) and [`ErrNotImplemented`](#errnotimplemented) —
override the level and exit code the caller asked for, because the error knows what kind
of ending it is and the reporting site usually does not. Everything else they carry —
the prefix, hints, details, the stack at debug, the help message and any `errors.Wrap`
context — is reported normally.

### Fatal

```go
func (h *StandardErrorHandler) Fatal(err error, prefixes ...string)
```

`Check(err, prefixes…, LevelFatal)`. Logs at error, then exits with the error's
[exit code](exit-codes.md). A nil `err` is a no-op and **does not exit** — `Fatal` is
not a way to force a successful termination.

### Error

```go
func (h *StandardErrorHandler) Error(err error, prefixes ...string)
```

`Check(err, prefixes…, LevelError)`. Logs at error; the process keeps running.

### Warn

```go
func (h *StandardErrorHandler) Warn(err error, prefixes ...string)
```

`Check(err, prefixes…, LevelWarn)`. Logs at warn; the process keeps running.

### How the variadic prefixes combine

`Fatal`, `Error` and `Warn` take `prefixes ...string` and concatenate them **with no
separator** before passing the result to `Check`. `Error(err, "cache", "update")`
produces `prefix=cacheupdate`, not `cache update`. Put any spacing or punctuation you
want in the strings themselves — `Error(err, "cache: ")` is the idiom used in this
module's own tests.

Passing no prefixes yields an empty string, and the field is then omitted.

### SetUsage

```go
func (h *StandardErrorHandler) SetUsage(usage func() error)
```

Registers the function that prints usage when an error wrapping
[`ErrRunSubCommand`](#errrunsubcommand) is reported. Keeping this a plain `func() error`
is what lets the module stay free of any CLI-framework dependency; with Cobra, pass
`cmd.Usage`.

- The handler stores **one** function. The last call wins, which is why it belongs in a
  per-command pre-run hook rather than being set once at the root — see
  [Print usage for a parent command](../how-to/print-usage.md#set-it-per-command-not-once-globally).
- **The error the usage function returns is discarded.** A printer that fails does so
  silently; the "Subcommand required" warning is still logged.
- Leaving it unset is valid. `ErrRunSubCommand` is still treated as handled, just with
  no usage output.

## Attaching information to an error

Each of these is a thin wrapper over `cockroachdb/errors`. They exist for
discoverability and a consistent house style, not encapsulation — calling the
underlying function directly is equally correct. The three that take an existing error —
`WithUserHint`, `WithUserHintf` and `WrapWithHint` — **return nil when given nil**, so
they are safe to apply unconditionally in a return statement. `NewAssertionFailure`
takes no error; it constructs a new one.

### WithUserHint

```go
func WithUserHint(err error, hint string) error
```

`errors.WithHint`. Attaches a user-facing recovery suggestion, rendered as the
[`hints` field](log-fields.md#hints) at every level, debug or not.

### WithUserHintf

```go
func WithUserHintf(err error, format string, args ...any) error
```

`errors.WithHintf`. The `Printf`-style form. The formatted string is the hint; it is
not redacted or truncated.

### WrapWithHint

```go
func WrapWithHint(err error, msg string, hint string) error
```

`errors.WithHint(errors.Wrap(err, msg), hint)` — adds context to the message *and*
attaches advice, in one call. The wrap captures a stack frame; the hint does not.

### NewAssertionFailure

```go
func NewAssertionFailure(format string, args ...any) error
```

`errors.AssertionFailedf`. Marks an error as a *programming* bug rather than something
the user did. Reporting one takes a distinct path — see
[Report levels](levels.md#what-an-assertion-failure-does).

Note that `cockroachdb/errors` attaches its **own** generic hint to every assertion
failure ("You have encountered an unexpected error… please check the public issue
tracker…"). That text will appear in the [`hints` field](log-fields.md#hints) of the
report whether or not you attach a hint of your own, and it is not configurable from
here.

## Exit codes

### WithExitCode

```go
func WithExitCode(err error, code int) error
```

Attaches a process exit code to `err` without altering its message or identity. Returns
`nil` for a nil `err` — attaching a code to "no error" cannot invent a failure.

The wrapper is transparent to `errors.Is` and `errors.As`, and survives further
wrapping. `code` is not validated: any `int` is accepted, including values outside the
0–255 range a POSIX process can actually return, in which case the shell sees the
low 8 bits.

### ExitCode

```go
func ExitCode(err error) int
```

Reads the code back.

| Argument | Result |
|---|---|
| `nil` | `0` |
| An error with no attached code | `1` |
| An error with a code attached | that code |
| A code attached more than once | the **outermost**, matching `errors.As` order |

It traverses joined errors as well as wraps, so a code attached to one branch of an
`errors.Join` is still found — but see
[Limitations](limitations.md#hints-are-lost-when-errors-are-joined) for what that same
join does to hints.

## Sentinels

### ErrNotImplemented

```go
var ErrNotImplemented = errors.New("command not yet implemented")
```

A command stub. Reported at **warn** with the message `Command not yet implemented`,
whatever level was requested, because a stub is not a failure.

### NewErrNotImplemented

```go
func NewErrNotImplemented(issueURL string) error
```

Builds an unimplemented error carrying `issueURL` as a cockroachdb issue link, which
the handler emits as a separate `INFO` line: `Track progress url=<issueURL>`.

- **`issueURL` is not validated.** An empty string produces `Track progress url=""`; a
  non-URL string is printed as given.
- **The result is not `errors.Is(err, ErrNotImplemented)`.** It is detected by
  `errors.HasUnimplementedError` instead. Both reach the same handler branch, but code
  of your own that matches on the sentinel will not match errors from this constructor.

### ErrRunSubCommand

```go
var ErrRunSubCommand = WithOutcome(
	errors.NewSentinel("errorhandling.run_subcommand", "subcommand required"),
	Outcome{Code: ExitCodeUsage, Level: slog.LevelWarn, Usage: true},
)
```

A parent command invoked with no subcommand, **where that is itself the mistake**. The
handler calls the [`SetUsage`](#setusage) printer, logs at warn, and the process exits
[`ExitCodeUsage`](exit-codes.md).

Returning it is a choice, not a rule. A parent that only groups its children can
equally treat a bare invocation as a request for help and succeed; reach for this when
the command genuinely cannot act on its own.

Detection is `errors.Is`, and a wrap's own message **is** reported —
`errors.Wrap(ErrRunSubCommand, "config")` logs `config: subcommand required`. The
[`Outcome`](#outcome) travels through the wrap with it.

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
usage, and exits `2`.

## Supplying a support channel

### HelpConfig

```go
type HelpConfig interface {
	SupportMessage() string
}
```

The extension point for a support-channel message. The module ships the interface and
no implementations, deliberately — see
[Add a support channel](../how-to/support-channel.md#why-the-module-ships-no-implementations).

- `SupportMessage` is called **on every reported error** that reaches the normal path,
  so it must be cheap. Resolve the message once at startup and return the stored value.
- Returning `""` suppresses the [`help` field](log-fields.md#help) — that is the
  designed way to stay silent when a channel is not configured.
- Passing `nil` for the whole interface disables help output without any nil handling
  on your side.
- It is called on the reporting goroutine with no context and no cancellation. It must
  not block, and it must be safe to call concurrently.

## Log field key constants

`KeyStacktrace`, `KeyHints`, `KeyDetails` and `KeyHelp` hold the field names the
handler uses. They are exported so tests can assert on a key without hard-coding the
string. See [Log fields](log-fields.md) for what each field contains and when it is
present.
