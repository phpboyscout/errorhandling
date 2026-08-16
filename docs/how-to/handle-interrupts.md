# Handle interrupts quietly

A user pressing <kbd>Ctrl</kbd>+<kbd>C</kbd> has not encountered a failure — they have
made a decision. Reporting that as `ERROR interrupted by signal` is noise: it looks
like your tool broke when it did exactly what it was told.

But the run must still **exit non-zero**, because that is how the shell, CI, and any
supervisor learn the run did not complete.

[`Quietly()`](../reference/api.md#quietly) resolves the tension: **report exactly like
`Fatal`, but log at debug.**

## Report an interrupt

```go
err := errorhandling.WithExitCode(
	errors.Newf("interrupted by signal: %v", sig),
	128+int(sig.(syscall.Signal)),
)

os.Exit(handler.Fatal(ctx, err, errorhandling.Quietly()))
```

The process exits `130` for SIGINT (`128+2`) or `143` for SIGTERM (`128+15`) — the
convention shells and supervisors understand — while the notice stays out of the user's
face. It is still there under debug, so anyone diagnosing a run sees it.

The general rule: reach for `Quietly()` whenever a termination is **expected and
user-initiated**, and **the exit code is itself the signal** — an error line would add
nothing.

Note it is read on the `Fatal` path only: `Error(ctx, err, Quietly())` still logs at
`ERROR`. And an error carrying an [`Outcome`](../reference/api.md#outcome) overrides it,
so if you want a quiet interrupt, do not also give it an outcome that says otherwise.

## Wiring a signal watcher

The module handles *reporting*; catching signals is your program's job. A pattern worth
copying, learned the hard way:

```go
sigs := make(chan os.Signal, 2) // buffered for BOTH the graceful and force signals
signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
defer signal.Stop(sigs)

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
	sig := <-sigs
	slog.Default().Warn("interrupt received — press again to force quit")
	cancel()           // first signal: graceful shutdown

	<-sigs
	os.Exit(130)       // second signal: force quit; a hung cleanup can't trap the user
}()
```

Three details that matter:

- **Buffer the channel to 2.** An unbuffered channel can drop the second signal — the
  one the user sends *because* the first didn't work.
- **A second signal must force-exit.** Otherwise a wedged cleanup traps them, and the
  only escape is `kill -9`.
- **Register both `os.Interrupt` and `syscall.SIGTERM`** with no build tags. On Windows
  only `os.Interrupt` ever fires; the SIGTERM registration is harmless.

`signal.NotifyContext` looks like the obvious tool here and isn't quite enough: it
cannot tell you **which** signal fired (you need that for `128+signum`) and cannot
observe a **second** signal (you need that for force-quit). `signal.Notify` plus
`context.WithCancel` gives both.

## The signal wins over the returned error

When a run is interrupted, the command tree usually returns `context.Canceled` — an
artefact of the cancellation, not the reason the run ended. Prefer the signal:

```go
err := runCommandTree(ctx)

if sig := receivedSignal(); sig != nil {
	flush()                                    // see below — before you exit

	return handler.Fatal(ctx, interruptErr(sig), errorhandling.Quietly())
}

if err != nil {
	return handler.Fatal(ctx, err)
}

return 0
```

Both branches return a code for `main` to act on, rather than ending the process where
they stand.

!!! danger "Flush before you exit, not in a defer"
    `Fatal` returns rather than exiting, so deferred cleanup in the *calling* frames does
    still run — but only up to whatever frame calls `os.Exit` with the code it returned.
    Anything deferred above that never runs. This is the classic way a telemetry flush
    silently stops happening the moment a real error occurs.

    Invoke pre-exit work **explicitly before** you exit, guarded by a `sync.Once` so the
    normal paths can still `defer` it — and give it a **fresh, bounded context**, never
    the one the interrupt just cancelled, or it aborts instantly.

## Interactive prompts and Ctrl-C

If your tool renders a TUI prompt, the terminal is usually in **raw mode** — there,
<kbd>Ctrl</kbd>+<kbd>C</kbd> arrives as a *keystroke*, not a signal. The prompt aborts;
your outer watcher never fires. That is the behaviour users expect: a stray Ctrl-C
cancels the question, not the whole run.

An **external** `kill -INT` is different: Go delivers it to every subscriber, so the
prompt aborts *and* the run cancels with 130. That is the right semantic for
supervisors — an external terminate should end the run, prompt or not.

## Related

- [Control the exit code](exit-codes.md)
- [The reporting model](../explanation/reporting-model.md#levels)
- [Report levels](../reference/levels.md#quietly-only-applies-to-fatal)
