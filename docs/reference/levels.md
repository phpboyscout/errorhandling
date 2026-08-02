# Report levels

The `level` argument to [`Check`](api.md#check) decides two things independently: the
`slog` level the report is logged at, and whether the process exits afterwards. The
level constants are plain strings, so an unrecognised value is possible and has a
defined behaviour.

## What each level does to an ordinary error

| Constant | Value | Logged at | Exits? | With what code |
|---|---|---|---|---|
| `LevelFatal` | `"fatal"` | `ERROR` | yes | [`ExitCode(err)`](exit-codes.md) |
| `LevelFatalQuiet` | `"fatal-quiet"` | `DEBUG` | yes | [`ExitCode(err)`](exit-codes.md) |
| `LevelError` | `"error"` | `ERROR` | no | — |
| `LevelWarn` | `"warn"` | `WARN` | no | — |
| anything else | — | `ERROR` | no | — |

"Ordinary" means anything that is not a special error or an assertion failure; those
two are covered further down and behave differently.

### LevelFatal

Logs the error message at `ERROR` with all the [fields](log-fields.md) the error
carries, then calls the exit function with the error's exit code.

### LevelFatalQuiet

Identical to `LevelFatal` in every way except the log level: the message goes to
`DEBUG`. It exists for terminations that are expected and user-initiated — a
`SIGINT`, a `SIGTERM` — where the non-zero exit code is the signal and an error line
would be noise. Turn debug on and the message is still there.

This is the only level whose output most users will never see, which is the point.
It is not a way to suppress a real failure: use it only when the exit code alone
carries the whole meaning. See
[Handle interrupts quietly](../how-to/handle-interrupts.md).

### LevelError

Logs at `ERROR` and returns. The caller keeps control and the process keeps running.
This is the right level for a failure that is worth telling someone about but does not
end the run — one file in a batch, one retry that did not work.

### LevelWarn

Logs at `WARN` and returns. For conditions that are not failures but are worth
surfacing: a deprecated flag, a fallback that had to be taken.

### What an unrecognised level does

Logs at `ERROR` and does not exit. This is deliberate: a typo in a level string must
never make a failure invisible, so the fallback is the loudest non-terminating option
rather than a silent drop or a panic.

There is no way to find out that a level was unrecognised — the report looks exactly
like `LevelError`. If you build level strings dynamically, validate them against the
constants yourself.

## Which levels terminate the process

Only `LevelFatal` and `LevelFatalQuiet`. Everything else — including an unrecognised
level — logs and returns.

A nil error terminates nothing at any level: `Check(nil, "", LevelFatal)` is a complete
no-op and does **not** exit zero.

## What a special error does, at every level

A **special error** is one the handler recognises structurally rather than reporting
verbatim:

- anything satisfying `errors.Is(err, ErrRunSubCommand)`
- anything satisfying `errors.Is(err, ErrNotImplemented)` or
  `errors.HasUnimplementedError(err)` — which includes everything from
  [`NewErrNotImplemented`](api.md#newerrnotimplemented)

These get a fixed presentation *before* the level is consulted, so the level only
decides whether the process then exits:

| Error | Always logged | Exits at `fatal` / `fatal-quiet` | Exits at other levels |
|---|---|---|---|
| `ErrRunSubCommand` | usage printed, then `WARN Subcommand required` | yes, code [`2`](exit-codes.md#exitcodeusage) | no |
| unimplemented | `WARN Command not yet implemented`, plus `INFO Track progress` when an issue link is attached | yes, code [`2`](exit-codes.md#exitcodeusage) | no |

Three consequences worth knowing before you rely on them:

- **The exit code is always `2`.** A code attached with
  [`WithExitCode`](api.md#withexitcode) is ignored on this path.
- **`LevelFatalQuiet` is not quiet here.** The `WARN` presentation is emitted whatever
  the level, so a special error reported quietly still prints a line.
- **The rest of the report is dropped** — prefix, hints, details, stack trace and help
  message all go unrendered. See
  [Limitations](limitations.md#special-errors-discard-most-of-the-report).

## What an assertion failure does

An error created by [`NewAssertionFailure`](api.md#newassertionfailure), or any error
satisfying `errors.HasAssertionFailure`, is reported **twice**:

1. `ERROR Internal error (assertion failure)` with the whole error value under the key
   `error`. This line is emitted at error level regardless of the requested level and
   regardless of whether debug is enabled.
2. `DEBUG Assertion detail` carrying the full `%+v` rendering under
   [`stacktrace`](log-fields.md#stacktrace) — only when the logger has debug enabled.
3. Then the error falls through to the ordinary path for the requested level, producing
   a second line with the error's own message and fields.

So an assertion failure at `LevelWarn` still produces an `ERROR` line, and one at
`LevelFatalQuiet` is not quiet. The reasoning is that an assertion failure means the
program is wrong, and the caller's opinion about severity is not the last word on that.

**How much of the stack that first line prints depends on your `slog` handler.**
The standard `TextHandler` formats an arbitrary value with `%+v`, which for a
`cockroachdb/errors` value means the message *and* every stack frame — at error level,
with debug off. The standard `JSONHandler` special-cases errors and prints
`err.Error()` alone. If you need assertion failures to stay one line at error level,
choose a handler that renders error values by message.

## Related

- [Exit codes](exit-codes.md) — which code the process actually uses
- [Log fields](log-fields.md) — what appears alongside the message
- [The reporting model](../explanation/reporting-model.md) — why the split is drawn here
