# Exit codes

The exit code a process ends with is decided in one place — the handler's fatal path —
from information attached to the error. This page states exactly which code wins.

For how to *use* this, see [Control the exit code](../how-to/exit-codes.md).

## How the exit code is resolved

Read top to bottom; the first row that matches decides.

| Situation | Code used |
|---|---|
| `err` is nil | nothing happens — the process does not exit |
| Level is not `fatal` or `fatal-quiet` | nothing happens — the process does not exit |
| `err` is a [special error](levels.md#what-a-special-error-does-at-every-level) | **`2`** (`ExitCodeUsage`), *even if a code was attached* |
| A code was attached with [`WithExitCode`](api.md#withexitcode) | that code — the outermost, if several |
| Anything else | **`1`** |

The third row is the one that surprises people: on the special-error path the handler
exits with `ExitCodeUsage` directly, without consulting the error. `Fatal(WithExitCode(
ErrRunSubCommand, 64))` exits `2`, not `64`.

## ExitCodeUsage

```go
const ExitCodeUsage = 2
```

The code used when a fatal-level report turns out to be a usage problem — a parent
command invoked with no subcommand, or a command that is not implemented yet. `2` is
the conventional Unix "command misuse" status, distinct from the general failure code
`1`, so a calling script can tell an invalid invocation from a run that started
properly and then failed.

It is a constant, not a default: there is no option to change it.

## Codes this package assigns on its own

Only two, and neither can be configured:

| Code | When |
|---|---|
| `1` | A fatal error with nothing attached |
| `2` | A fatal [special error](levels.md#what-a-special-error-does-at-every-level) |

Everything else is a code you attached. There is no built-in taxonomy, no mapping from
error kind to code, and no validation — see
[Conventional codes](../how-to/exit-codes.md#conventional-codes) for the conventions
worth honouring.

## What is not checked

- **The range.** `WithExitCode(err, 300)` is accepted; the operating system takes the
  low 8 bits, so the shell reports `44`. Keep codes within 0–255.
- **Zero.** `WithExitCode(err, 0)` is accepted and exits `0`, reporting success from a
  path that had an error. Nothing warns about this.
- **Negative values.** Also accepted, and also truncated by the OS.

## When the attached code does not survive

`WithExitCode` wraps the error in an unexported type that `errors.Is`, `errors.As` and
`errors.Join` all see through. Two things lose it:

- **Encoding and decoding across a process boundary.** `cockroachdb/errors` can
  serialise an error and rebuild it elsewhere, but the exit-code wrapper is not a
  registered type, so a round trip through `errors.EncodeError`/`errors.DecodeError`
  comes back reporting `1`.
- **Rebuilding the error from its message.** Anything that reconstructs an error from
  `err.Error()` — a retry layer, an RPC boundary that only carries strings — drops the
  wrapper along with the hints and the stack.

Attach the code at the point the error reaches the process that will exit, if it has to
cross a boundary in between.

## Reading a code back in a test

```go
errorhandling.ExitCode(nil)                                    // 0
errorhandling.ExitCode(errors.New("boom"))                     // 1
errorhandling.ExitCode(errorhandling.WithExitCode(err, 130))   // 130
```

To assert on the code the *handler* would use, inject the exit function rather than
calling `ExitCode` — the special-error rule above lives in the handler, not in
`ExitCode`, so the two can legitimately disagree. See
[Test error handling](../how-to/test-error-handling.md#never-let-fatal-kill-your-tests).

## Related

- [Report levels](levels.md) — which levels exit at all
- [Control the exit code](../how-to/exit-codes.md) — the guide
- [Handle interrupts quietly](../how-to/handle-interrupts.md) — the `128+signum` convention
