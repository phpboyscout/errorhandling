# The reporting model

This page explains the shape of the package: what belongs to the error, what belongs to
the report, and why the split is drawn where it is.

## Two concerns, deliberately separate

**Creating** an error and **reporting** one are different jobs, done in different
places, by different code:

| | Creation | Reporting |
|---|---|---|
| Who | the code that detected the problem, anywhere in your tree | one place, near the top |
| With what | [`cockroachdb/errors`](https://github.com/cockroachdb/errors) + the helpers here | `ErrorHandler` |
| Produces | an error carrying context, a hint, a stack, maybe an exit code | log output, and possibly process exit |

Everything below follows from keeping those apart. Deep code enriches an error and
returns it; it does not decide the program's fate. One place near the top decides.

## Bubble up; don't exit deep

**Report errors at the top, not where they happen.** A `Fatal` (or a bare `os.Exit`)
buried in business logic terminates the process *immediately* — deferred functions do
not run, so buffered writes are lost, temp files survive, and connections close
uncleanly.

```go
// ✅ return the error; let the caller decide
func doWork(ctx context.Context) error {
	if err := step(); err != nil {
		return errors.Wrap(err, "step failed")
	}

	return nil
}

// ❌ decides the whole program's fate from deep inside it
func doWork(ctx context.Context) {
	handler.Fatal(step())
}
```

Library-ish code should create and wrap errors and let its caller judge how fatal they
are. The same error might be fatal to one command and a warning to another — only the
caller knows.

## Levels

| Level | Behaviour |
|---|---|
| `LevelFatal` | log at error, then exit with the error's [exit code](../how-to/exit-codes.md) |
| `LevelFatalQuiet` | exit **identically**, but log at *debug* — for expected terminations like [SIGINT](../how-to/handle-interrupts.md) |
| `LevelError` | log at error; keep running |
| `LevelWarn` | log at warn; keep running |

`Fatal` / `Error` / `Warn` are convenience wrappers over `Check(err, prefix, level)`.
An unrecognised level deliberately falls back to logging at **error** rather than
silently swallowing the error — a reporting bug must never make a failure invisible.

`Check(nil, …)` is a no-op, so `handler.Check(doThing(), "", LevelError)` is safe
without a preceding nil test.

## What gets rendered, and when

A report is assembled from the error itself:

- **The message** — always.
- **Hints** — always, when present. A hint is the *actionable* half of an error and is
  useless if it only appears in debug mode.
- **Stack traces and details** — only when the logger has **debug enabled**. This is
  decided by asking the logger (`Enabled(ctx, slog.LevelDebug)`), not by reading a
  level off a config, so whatever your slog handler considers debug is what counts.
- **A support message** — when a [`HelpConfig`](../how-to/support-channel.md) is
  supplied and returns a non-empty string.

The prefix argument is **not** prepended to the message. It is attached as a structured
`prefix` attribute, so it appears as `prefix=cache-update` alongside the message rather
than being baked into the text — greppable, and it doesn't corrupt the message for
anything matching on it.

## The exit code belongs to the error

A failure knows its own severity at the point it is detected, but the process exits
somewhere else entirely. Rather than thread an exit code through every return, attach
it to the error and let it travel:

```go
return errorhandling.WithExitCode(err, 3)
```

The attachment is transparent — `errors.Is` and `errors.As` still see through it, and
it survives further wrapping — so nothing downstream has to know it's there. At the
top, the fatal path reads it back. `ExitCode(nil)` is `0`, and an error with nothing
attached is `1`.

The payoff is architectural: **one exit path**. No `os.Exit` scattered through the
tree, so there's exactly one place where the process can die and exactly one place to
change if that behaviour needs to.

## Sentinels that mean something

Two error values trigger behaviour rather than just being reported:

- **`ErrNotImplemented`** — reported as a *warning*, not an error: a stub is not a
  failure. `NewErrNotImplemented(issueURL)` attaches a tracking link, which the handler
  surfaces so a user can follow progress rather than just being told "no".
- **`ErrRunSubCommand`** — a parent command invoked with no subcommand. The handler
  prints usage through the [`SetUsage`](../how-to/print-usage.md) seam and warns. Not
  finding a subcommand is a usage problem, so the answer is to *show the usage*.

Both are presented (usage / notice, at warning level) regardless of the report level.
Reported at **fatal** level they still terminate the process, but with the conventional
usage exit code [`2` (`ExitCodeUsage`)](../how-to/exit-codes.md#special-errors-exit-2)
rather than the generic `1`, so an invalid invocation is not mistaken for success.

## Assertion failures are different

`NewAssertionFailure` marks an *internal* invariant breach — "this should be
impossible" — rather than a user-facing problem. Those are reported as internal errors
and carry their detail into debug output. The distinction matters because the two
audiences differ: a user can act on "config file not found"; nobody outside your team
can act on a violated invariant.

Note that an assertion failure still falls through to normal reporting afterwards,
rather than short-circuiting — the failure is surfaced *and* logged through the usual
path.

## Why an interface

`ErrorHandler` is an interface so the reporting boundary can be substituted. Tests use
[the published mock](../how-to/test-error-handling.md) to assert *that* something was
reported without parsing log output; an application can swap in its own implementation
entirely. The concrete `StandardErrorHandler` remains exported for callers that want
to construct it directly.

## Related

- [Why cockroachdb/errors](why-cockroachdb-errors.md)
- [Write actionable errors](../how-to/actionable-errors.md)
- [Control the exit code](../how-to/exit-codes.md)
