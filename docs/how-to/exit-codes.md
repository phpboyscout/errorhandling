# Control the exit code

Scripts, CI jobs, and supervisors branch on exit codes. The code that *knows why* a run
failed is usually deep in your program, but the process exits at the top — so attach
the code to the error and let it travel.

## Attach a code

```go
return errorhandling.WithExitCode(err, 3)
```

Report it at fatal level and the handler exits with that code:

```go
handler.Fatal(err) // exits 3
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

## Why not just call os.Exit?

Because `os.Exit` **skips every pending `defer`**. An exit buried in your call tree
loses buffered writes, leaves temp files behind, and closes connections uncleanly — and
it makes the surrounding code untestable, because a test that reaches it kills the test
binary.

Attaching the code to the error keeps the *decision* where the knowledge is and the
*act* of exiting in exactly one place. One exit path is also one place to change when
that behaviour needs to.

!!! danger "Fatal exits — so it also skips defers"
    The same caveat applies to the handler's own fatal path: `Fatal` (and
    `Check(..., LevelFatal)`) call the exit func, so **deferred cleanup in the calling
    frames does not run.**

    Anything that *must* happen before the process dies — flushing telemetry, removing
    a lock file — has to be invoked **explicitly before** the fatal call, not left to a
    `defer`. Make it idempotent (a `sync.Once`) so the non-fatal paths can still defer
    it safely:

    ```go
    flush := sync.OnceFunc(func() { telemetry.Flush(context.Background()) })
    defer flush() // normal paths

    if err := run(); err != nil {
        flush()          // explicitly, before the exit
        handler.Fatal(err)
    }
    ```

    Use a **fresh, bounded** context for that cleanup — never a context that the
    failure itself already cancelled, or the flush aborts immediately.

## Testing without killing the test binary

Inject the exit function — see [testing](test-error-handling.md):

```go
var code int
h := errorhandling.New(logger, nil,
	errorhandling.WithExitFunc(func(c int) { code = c }))

h.Fatal(errorhandling.WithExitCode(errors.New("boom"), 3))
assert.Equal(t, 3, code)
```

## Related

- [Handle interrupts quietly](handle-interrupts.md)
- [The reporting model](../explanation/reporting-model.md#the-exit-code-belongs-to-the-error)
- [Test error handling](test-error-handling.md)
