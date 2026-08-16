# The reporting model

This page explains the shape of the package: what belongs to the error, what belongs to
the report, and why the split is drawn where it is.

## Two concerns, deliberately separate

**Creating** an error and **reporting** one are different jobs, done in different
places, by different code:

| | Creation | Reporting |
|---|---|---|
| Who | the code that detected the problem, anywhere in your tree | one place, near the top |
| With what | [`gitlab.com/phpboyscout/go/errors`](https://errors.go.phpboyscout.uk) | `ErrorHandler` |
| Produces | an error carrying context, a hint, a stack, maybe an exit code | log output, and possibly process exit |

Everything below follows from keeping those apart. Deep code enriches an error and
returns it; it does not decide the program's fate. One place near the top decides.

## Bubble up; don't exit deep

**Report errors at the top, not where they happen.** A bare `os.Exit` buried in business
logic terminates the process *immediately* — deferred functions do not run, so buffered
writes are lost, temp files survive, and connections close uncleanly.

This module will not do that to you: `Fatal` reports and *returns* a code. But a `Fatal`
call deep in a call tree still decides the program's fate from a place that cannot know
it, and the code it returns is then thrown away by a caller that has nothing to do with
it.

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
	handler.Fatal(ctx, step())
}
```

Library-ish code should create and wrap errors and let its caller judge how fatal they
are. The same error might be fatal to one command and a warning to another — only the
caller knows.

## Levels

| Method | Behaviour |
|---|---|
| `Fatal` | log at error, **return** the error's [exit code](../how-to/exit-codes.md) |
| `Error` | log at error; keep running |
| `Warn` | log at warn; keep running |

There is no level *argument* and no level constants: the method is the level. The one
modifier is [`Quietly()`](../reference/api.md#quietly), which demotes a fatal report to
debug for an expected termination like [SIGINT](../how-to/handle-interrupts.md) without
changing the code.

A nil error is a no-op at all three, so `handler.Error(ctx, doThing())` is safe without a
preceding nil test.

## What gets rendered, and when

A report is assembled from the error itself:

- **The message** — always.
- **Hints** — always, when present. A hint is the *actionable* half of an error and is
  useless if it only appears in debug mode.
- **Details** — always, alongside the hints. They are a separate *category* of
  information, not a privacy boundary.
- **Stack traces** — only when the logger has **debug enabled**. This is decided by
  asking the logger (`Enabled(ctx, slog.LevelDebug)`), not by reading a level off a
  config, so whatever your slog handler considers debug is what counts.
- **A support message** — when a [`HelpConfig`](../how-to/support-channel.md) is
  supplied and returns a non-empty string.

That list holds for **every** error, sentinels included. The complete field-by-field
account is in [the log fields reference](../reference/log-fields.md).

The prefix is **not** prepended to the message. It is attached as a structured
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

## The outcome belongs to the error too

Some errors know not just their severity but how the program should *end*. A parent
command invoked with no subcommand should print usage, warn rather than shout, and exit
`2`. None of that is knowable at the reporting site, which sees only "an error".

So it travels with the error, the same way the exit code does:

```go
var ErrRunSubCommand = WithOutcome(
	errors.NewSentinel("errorhandling.run_subcommand", "subcommand required"),
	Outcome{Code: ExitCodeUsage, Level: slog.LevelWarn, Usage: true},
)
```

An [`Outcome`](../reference/api.md#outcome) says what code to end with, how loudly to
report, optionally what message to use instead of the error's own, and whether to print
usage first. The handler reads it and obeys — it does not switch on which error it is.

That last part is the design. This module used to hold a closed `switch` over three
sentinels it happened to know about, which meant a consumer could not have an error of
its own that ended the program politely — the canonical case being a completed
self-update, which must stop the run with exit `0` and no error line. An outcome is
declared beside the error that needs it, by whoever needs it.

The three sentinels here are simply the ones this module ships:

- **[`ErrNotImplemented`](../reference/api.md#errnotimplemented)** — a stub is not a
  failure, so it warns rather than errors. `NewErrNotImplemented(issueURL)` attaches a
  tracking link so a user can follow progress rather than just being told "no".
- **[`ErrRunSubCommand`](../reference/api.md#errrunsubcommand)** — no subcommand was
  given, and that is the mistake. Usage is the answer, so usage is what it prints.
- **[`ErrUnknownSubCommand`](../reference/api.md#errunknownsubcommand)** — a subcommand
  *was* given and does not exist. Same remedy, different message, and a distinct kind so
  the two can be told apart.

Because an outcome is an attachment rather than a special case, **everything else the
error carries is still reported**: wrap one with context and the wrap's message is what
you read, with its hints, details, prefix and stack intact. The only things it overrides
are the level and the code, which is exactly what it exists to state.

## Assertion failures are different

`NewAssertionFailure` marks an *internal* invariant breach — "this should be
impossible" — rather than a user-facing problem. Those are reported as internal errors
and carry their detail into debug output. The distinction matters because the two
audiences differ: a user can act on "config file not found"; nobody outside your team
can act on a violated invariant.

It is reported like any other error — **one record**, at the level you asked for —
identified by the kind `errorhandling.assertion_failure` inside the
[`err` group](../reference/log-fields.md#err).

Earlier versions emitted a second, fixed `ERROR Internal error (assertion failure)` line
regardless of the requested level, on the reasoning that a broken invariant is not
something the caller should be able to downplay. That line is gone. The kind says the
same thing in a field, which a query can filter on and a log prefix cannot — and the
duplicate record made every assertion failure two entries to correlate.

Nothing stops you reporting one at `Warn`. If a broken invariant should always be loud
in your tool, attach an [`Outcome`](../reference/api.md#outcome) that says so, or report
it at `Error` — the module no longer overrides that judgement on your behalf.

## Why an interface

`ErrorHandler` is an interface so the reporting boundary can be substituted. Tests use
[the published mock](../how-to/test-error-handling.md) to assert *that* something was
reported without parsing log output; an application can swap in its own implementation
entirely. The concrete `StandardErrorHandler` remains exported for callers that want
to construct it directly.

## Related

- [Why go/errors](why-go-errors.md)
- [Write actionable errors](../how-to/actionable-errors.md)
- [Control the exit code](../how-to/exit-codes.md)
- [Report levels](../reference/levels.md) — the same model as a lookup table
- [Limitations](../reference/limitations.md) — where the model stops
