# Exit codes

The exit code is decided in one place — [`Fatal`](api.md#fatal) — from information
attached to the error. This page states exactly which code wins.

**`Fatal` returns the code; it does not exit.** Acting on it is `main`'s job. A library
that called `os.Exit` would skip every deferred cleanup between itself and `main`.

For how to *use* this, see [Control the exit code](../how-to/exit-codes.md).

## How the exit code is resolved

Read top to bottom; the first row that matches decides.

| Situation | Code returned |
|---|---|
| `err` is nil | `0`, and nothing is logged |
| `err` carries an [`Outcome`](api.md#outcome) | the outcome's `Code`, *even if a code was attached* |
| A code was attached with [`WithExitCode`](api.md#withexitcode) | that code — the outermost, if several |
| Anything else | `1` |

The second row is the one that surprises people. An outcome is the more specific
statement about how the error ends, so it beats an attached code:
`Fatal(ctx, WithExitCode(ErrRunSubCommand, 64))` returns `2`, not `64`.

[`Error`](api.md#error) and [`Warn`](api.md#warn) return nothing at all — they are not
terminal, so there is no code to resolve.

## ExitCodeUsage

```go
const ExitCodeUsage = 2
```

The code carried by this module's three usage sentinels — a parent invoked with no
subcommand, a parent given a verb it does not have, and a command that is not
implemented yet. `2` is the conventional Unix "command misuse" status, distinct from the
general failure code `1`, so a calling script can tell an invalid invocation from a run
that started properly and then failed.

It is a constant, not a default: there is no option to change it. What *is* adjustable is
which errors carry it — [`WithOutcome`](api.md#withoutcome) puts it on an error of your
own.

## Codes this package assigns on its own

| Code | When |
|---|---|
| `0` | `Fatal(ctx, nil)`, or an outcome that is terminal and successful |
| `1` | A fatal error with no outcome and no attached code |
| `2` | An error carrying a `ExitCodeUsage` outcome |

An outcome code of `0` is legitimate and meaningful: an error can be terminal *and*
successful. A completed self-update is exactly that — the run must stop, and nothing went
wrong.

Everything else is a code you attached. There is no built-in taxonomy, no mapping from
error kind to code, and no validation — see
[Conventional codes](../how-to/exit-codes.md#conventional-codes) for the conventions
worth honouring.

## What is not checked

- **The range.** `WithExitCode(err, 300)` is accepted; the operating system takes the
  low 8 bits, so the shell reports `44`. Keep codes within 0–255.
- **Zero.** `WithExitCode(err, 0)` is accepted and returns `0`, reporting success from a
  path that had an error. Nothing warns about this.
- **Negative values.** Also accepted, and also truncated by the OS.

## When the attached code does not survive

`WithExitCode` wraps the error in an unexported type that `errors.Is`, `errors.As` and
`errors.Join` all see through — [`ExitCode`](api.md#exitcode) finds a code through a
join.

What loses it is **rebuilding the error from its message**: anything that reconstructs an
error from `err.Error()` — a retry layer, an RPC boundary that carries only strings —
drops the wrapper along with the hints and the stack.

Attach the code at the point the error reaches the process that will exit, if it has to
cross a boundary in between.

## Reading a code back in a test

```go
errorhandling.ExitCode(nil)                                    // 0
errorhandling.ExitCode(errors.New("boom"))                     // 1
errorhandling.ExitCode(errorhandling.WithExitCode(err, 130))   // 130
```

`ExitCode` reports the *attached* code and knows nothing about outcomes, so it and
`Fatal` can legitimately disagree — `ExitCode(ErrRunSubCommand)` is `1` while
`Fatal(ctx, ErrRunSubCommand)` returns `2`. To assert on what the process would use,
assert on `Fatal`'s return value. It is a plain `int` and nothing exits, so no injection
is needed. See
[Test error handling](../how-to/test-error-handling.md).

## Related

- [Report levels](levels.md) — how loudly the same error is reported
- [Control the exit code](../how-to/exit-codes.md) — the guide
- [Handle interrupts quietly](../how-to/handle-interrupts.md) — the `128+signum` convention
