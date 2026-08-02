# Limitations

What this package does **not** do, what it does not check, and the combinations that do
not work. Each of these is current behaviour verified against the source, not a plan.

## Special errors discard most of the report

When an error satisfies `errors.Is(err, ErrRunSubCommand)`,
`errors.Is(err, ErrNotImplemented)` or `errors.HasUnimplementedError`, the handler
prints a fixed presentation and returns. Everything else the error was carrying is
dropped:

- the prefix
- hints, however they were attached
- details and the stack trace, even with debug on
- the `HelpConfig` support message
- the error's own message, including any context added by `errors.Wrap`

`errors.Wrap(ErrRunSubCommand, "config")` reports exactly `WARN Subcommand required` —
the word "config" never appears. If you need context on a usage failure, it has to go
into the usage output itself, not into the error.

## A special error's exit code cannot be chosen

A fatal special error always exits `2`. A code attached with `WithExitCode` is not
consulted on that path. There is no option to change `ExitCodeUsage` and no way to opt
a particular sentinel out of it.

## LevelFatalQuiet is not always quiet

`LevelFatalQuiet` demotes the *ordinary* report to debug. It does not suppress the two
reports that are emitted before the level is consulted:

- a [special error](#special-errors-discard-most-of-the-report) still logs its `WARN`
  presentation
- an [assertion failure](levels.md#what-an-assertion-failure-does) still logs
  `ERROR Internal error (assertion failure)`

So "quiet" holds for the case it was designed for — an interrupt carrying only an exit
code — and not in general.

## Hints are lost when errors are joined

Combining errors loses the guidance attached to them, and how much you lose depends on
which combinator you used:

| Combinator | Hints that survive |
|---|---|
| `errors.Wrap` | all of them |
| `errors.CombineErrors(a, b)` | only `a`'s |
| `errors.Join(a, b)` | **none** |

This is `errors.FlattenHints` behaviour in `cockroachdb/errors`, not something this
package can fix from the outside. If you fan out concurrent work and want the user to
see the advice, report each failure separately, or re-attach a single hint to the
combined error before reporting it.

Exit codes are unaffected — [`ExitCode`](api.md#exitcode) does traverse a join.

## Exit codes do not survive a process boundary

The exit-code wrapper is an unexported type and is not registered with
`cockroachdb/errors`' encoding, so an error that is encoded and decoded — across a gRPC
boundary, through a queue — arrives with no code attached and reports `1`. Anything that
rebuilds an error from its message string loses it too, along with the hints and the
stack.

## Fatal skips deferred cleanup

`Fatal`, and `Check` at either fatal level, call the exit function. `os.Exit` does not
run deferred functions, so buffered writes are lost and temp files survive. This is not
a bug in the package — it is what exiting means — but it catches people out.

Invoke anything that must happen before the process dies **explicitly, before** the
fatal call, guarded by a `sync.Once` so the normal paths can still defer it. See
[the worked pattern](../how-to/exit-codes.md#why-not-just-call-osexit).

## There is no output writer

All output goes through the `*slog.Logger` you supply. There is no writer option, no
colour control, no format control and no way to send usage output and error output to
different places from inside the handler. (An earlier `WithWriter` option existed and
was removed in v0.1.1 as dead code.)

To change where reports go, or how they look, construct a different `slog.Handler`.
To send *usage* somewhere specific, do it in the closure you pass to
[`SetUsage`](api.md#setusage).

## Fixed strings are not translatable

`Subcommand required`, `Command not yet implemented`, `Track progress` and
`Internal error (assertion failure)` are English literals in the source. There is no
message catalogue and no hook to replace them. A tool that needs localised output has
to catch those conditions before they reach the handler.

## Nothing is redacted

Messages, hints and details are passed to the logger verbatim. `cockroachdb/errors`
distinguishes safe from unsafe details, but this package does not use that distinction —
`errors.FlattenDetails` returns everything. Do not attach a credential, a token or
personal data to an error and expect the report to hide it.

## One usage printer, no per-command registry

The handler stores a single `func() error`. There is no map from command to printer and
no stack. Setting it once at the root means every parent command prints the root's
usage; the fix is to set it in each command's pre-run, which is the caller's job.

## The handler is not concurrency-safe to reconfigure

`SetUsage` writes a struct field with no lock, and every report reads `Logger`, `Help`,
`Exit` and `Usage` unguarded. Reporting from several goroutines is fine. Calling
`SetUsage` while another goroutine reports is a data race — the race detector will say
so. Configure the handler before you start concurrent work.

## Nil is not defended against

- A nil `*slog.Logger` passed to `New` panics on the first report, not at construction.
- A `StandardErrorHandler` built as a struct literal without `Exit` logs the fatal
  report and then panics instead of exiting.
- `WithExitFunc(nil)` is accepted and produces the same panic.

`New` with a non-nil logger is the supported construction path; the others are the
consequence of the struct being exported.

## No telemetry, no crash reporting, no panic recovery

The package logs and exits. It does not install a `recover`, does not build a Sentry
report — `cockroachdb/errors` can, but nothing here calls it — and emits no metrics or
traces. The dependency footprint is enforced by a guard test, so that is not going to
change quietly.

## No exit code is derived from the level

`LevelFatal` and `LevelFatalQuiet` exit with the same code for the same error. Severity
and exit status are separate decisions here: the level chooses how loudly to report,
the attached code chooses what the process returns.

## Related

- [Report levels](levels.md) — the behaviour these limits qualify
- [Exit codes](exit-codes.md) — the resolution order
- [Why cockroachdb/errors](../explanation/why-cockroachdb-errors.md) — what the
  foundation does and does not buy
