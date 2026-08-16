# Limitations

What this package does **not** do, what it does not check, and the combinations that do
not work. Each of these is current behaviour verified against the source, not a plan.

## An outcome overrides the level and code the caller asked for

An error carrying an [`Outcome`](api.md#outcome) decides its own log level and exit
code, whatever the reporting call said. `Error(ctx, ErrRunSubCommand)` logs at `WARN`,
and a code attached with [`WithExitCode`](api.md#withexitcode) loses to the outcome's.

That is the point of an outcome rather than a defect — the error knows what kind of
ending it is and the reporting site usually does not — but it does mean a caller cannot
promote one of these to an error-level report without replacing the outcome.

## Quietly is ignored by Error and Warn

[`Quietly`](api.md#quietly) is read on the [`Fatal`](api.md#fatal) path only.
`Error(ctx, err, Quietly())` logs at `ERROR`, and no error or warning tells you the
option did nothing.

It exists for a termination whose whole meaning is the exit code; a non-terminal report
has no code to carry the meaning instead.

## A usage outcome's exit code cannot be chosen

The three sentinels this module ships carry [`ExitCodeUsage`](exit-codes.md#exitcodeusage).
There is no option to change the constant, and no way to opt a particular sentinel out
of it. Attaching a different code to one of them does nothing — the outcome wins.

What you *can* do is build your own error with a different outcome:
`WithOutcome(err, Outcome{Code: 64, ...})`.

## Details are not debug-gated

Hints and details both appear in the [`err` group](log-fields.md#err) at every level,
debug or not. Details are a *category* of information, not a privacy boundary — if
something must not reach an ordinary user, it must not be attached to the error.

## Nothing is redacted

Messages, hints, details and attributes are passed to the logger verbatim. Do not attach
a credential, a token or personal data to an error and expect the report to hide it.

## Exit codes do not survive a rebuilt error

The exit-code wrapper is an unexported type. `errors.Is`, `errors.As` and `errors.Join`
all see through it, and [`ExitCode`](api.md#exitcode) finds a code through a join — but
anything that reconstructs an error from `err.Error()`, such as an RPC boundary carrying
only strings, drops it along with the hints and the stack.

## Fatal does not exit, and nothing else does either

`Fatal` reports and **returns** the code it believes the process should use. Nothing in
this module calls `os.Exit`.

That is a deliberate limitation on what the module will do for you: `main` owns
termination, because a library that exits skips every deferred cleanup between itself
and `main`. If nothing acts on the returned code, the process ends normally. See
[the worked pattern](../how-to/exit-codes.md#why-not-just-call-osexit).

## There is no output writer

All output goes through the `*slog.Logger` you supply. There is no writer option, no
colour control, no format control and no way to send usage output and error output to
different places from inside the handler.

To change where reports go, or how they look, construct a different `slog.Handler`. To
send *usage* somewhere specific, do it in the closure you pass to
[`SetUsage`](api.md#setusage).

## Fixed strings are not translatable

`subcommand required`, `unknown subcommand`, `command not yet implemented` and
`internal invariant violated` are English literals in the source. There is no message
catalogue and no hook to replace them.

A tool that needs localised output can override the text per report with an outcome's
`Message`, or catch the condition before it reaches the handler.

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
- A `StandardErrorHandler` built as a struct literal without a `Logger` does the same.

`New` with a non-nil logger is the supported construction path; the others are the
consequence of the struct being exported.

## No telemetry, no crash reporting, no panic recovery

The package logs and returns. It does not install a `recover`, does not build a crash
report, and emits no metrics or traces. The dependency footprint is enforced by a guard
test, so that is not going to change quietly.

## Related

- [Report levels](levels.md) — the behaviour these limits qualify
- [Exit codes](exit-codes.md) — the resolution order
- [Why go/errors](../explanation/why-go-errors.md) — what the foundation does and does
  not buy
