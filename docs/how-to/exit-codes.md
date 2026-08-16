# Control the exit code

Scripts, CI jobs, and supervisors branch on exit codes. The code that *knows why* a run
failed is usually deep in your program, but the process exits at the top — so attach
the code to the error and let it travel.

## Attach a code

```go
return errorhandling.WithExitCode(err, 3)
```

Report it with `Fatal` and that is the code you get back to exit on:

```go
os.Exit(handler.Fatal(ctx, err)) // 3
```

Nothing in between needs to know. The attachment is **transparent to `errors.Is` and
`errors.As`**, and it **survives further wrapping**, so hints, stack traces, and
sentinel matching all keep working:

```go
err := errorhandling.WithExitCode(ErrConfigMissing, 3)
err = errors.Wrap(err, "startup failed")

errors.Is(err, ErrConfigMissing)   // still true
errorhandling.ExitCode(err)        // still 3
```

## Read it back

```go
errorhandling.ExitCode(nil)                 // 0 — no error, no failure
errorhandling.ExitCode(errors.New("boom"))  // 1 — an error with nothing attached
errorhandling.ExitCode(coded)               // whatever was attached
```

`WithExitCode(nil, 3)` returns `nil` — attaching a code to "no error" cannot invent a
failure.

If a code is attached more than once, the **outermost wins**, matching `errors.As`
traversal order. So an outer layer can deliberately override an inner decision.

## Conventional codes

There is no enforced taxonomy — pick what suits your tool — but these conventions are
worth honouring because other software already understands them:

| Code | Meaning |
|---|---|
| `0` | success |
| `1` | general failure — the default when nothing is attached |
| `2` | conventionally, usage/CLI misuse |
| `128+signum` | terminated by a signal — `130` for SIGINT (Ctrl-C), `143` for SIGTERM |

The `128+signum` convention is what shells, CI runners, and supervisors expect from a
signalled process; see [handling interrupts](handle-interrupts.md).

## An outcome states the code, and wins

Some errors know how the program should end. This module's usage sentinels —
`ErrRunSubCommand`, `ErrUnknownSubCommand`, `ErrNotImplemented` — carry an
[`Outcome`](../reference/api.md#outcome) saying "warn, print usage, exit `2`":

```go
code := handler.Fatal(ctx, errorhandling.ErrRunSubCommand) // prints usage, code 2
```

This lets a calling script tell an *invalid invocation* apart from an ordinary runtime
failure, and matches what CLI frameworks such as Cobra do in the same situation.

An outcome beats an attached code: `Fatal(ctx, WithExitCode(ErrRunSubCommand, 64))`
returns `2`, not `64`, because the outcome is the more specific statement about how this
error ends. If a usage failure needs its own code, attach your own outcome rather than
wrapping the sentinel:

```go
errorhandling.WithOutcome(err, errorhandling.Outcome{
	Code: 64, Level: slog.LevelWarn, Usage: true,
})
```

The full resolution order is in
[the exit codes reference](../reference/exit-codes.md#how-the-exit-code-is-resolved).

## Why not just call os.Exit?

Because `os.Exit` **skips every pending `defer`**. An exit buried in your call tree
loses buffered writes, leaves temp files behind, and closes connections uncleanly — and
it makes the surrounding code untestable, because a test that reaches it kills the test
binary.

Attaching the code to the error keeps the *decision* where the knowledge is and the
*act* of exiting in exactly one place. One exit path is also one place to change when
that behaviour needs to.

**`Fatal` does not exit either.** It reports and returns a code; `main` is the one place
that acts on it:

```go
func main() {
	os.Exit(run())
}

func run() int {
	flush := sync.OnceFunc(func() { telemetry.Flush(context.Background()) })
	defer flush() // normal paths

	if err := doWork(); err != nil {
		flush() // explicitly, before returning a code main will exit on

		return handler.Fatal(context.Background(), err)
	}

	return 0
}
```

Deferred cleanup in `run` still runs, because `run` returns normally — but anything
deferred *above* `os.Exit` does not. Make pre-exit work idempotent (a `sync.Once`) so
the normal paths can still defer it, and give it a **fresh, bounded** context, never one
the failure itself already cancelled.

## Testing

`Fatal` returns an `int` and nothing exits, so there is no seam to inject — see
[testing](test-error-handling.md):

```go
h := errorhandling.New(logger, nil)

code := h.Fatal(t.Context(), errorhandling.WithExitCode(errors.New("boom"), 3))
assert.Equal(t, 3, code)
```

## What is not supported

- **Overriding the code on an error that carries an outcome.** Attach a different
  outcome instead — see above.
- **Mapping a level to a code.** `Quietly()` changes how loudly a fatal report is
  logged, never what it returns.
- **Carrying a code through a rebuilt error.** Anything that reconstructs an error from
  its message string drops the wrapper; the code comes back as `1`.
- **Range checking.** `WithExitCode(err, 300)` is accepted and the shell sees `44`.

## Related

- [Handle interrupts quietly](handle-interrupts.md)
- [The reporting model](../explanation/reporting-model.md#the-exit-code-belongs-to-the-error)
- [Test error handling](test-error-handling.md)
- [Exit codes reference](../reference/exit-codes.md)
