# errorhandling

**Turn an error into something a user can act on.** Attach a hint that says what to
*do*; attach the exit code to the error itself; report it once, at the top; and keep
stack traces for when someone asks for them.

```bash
go get gitlab.com/phpboyscout/go/errorhandling
```

`gitlab.com/phpboyscout/go/errorhandling` is the error-reporting layer extracted from
[go-tool-base](https://gitlab.com/phpboyscout/go-tool-base). It wraps
[cockroachdb/errors](https://github.com/cockroachdb/errors) with a small pipeline:
levels, hints, exit codes, debug-gated diagnostics, and an optional support-channel
message.

## Why

- **Two jobs, kept separate.** *Creating* an error (with context, hints, and a stack)
  is `cockroachdb/errors`' job. *Reporting* one — deciding the level, the exit code,
  and what the user sees — is this package's. See
  [the reporting model](explanation/reporting-model.md).
- **The error carries its exit code.** `WithExitCode(err, 3)` travels with the error
  through every wrap, so the code that knows *why* something failed decides how the
  process exits — and every exit stays on one path instead of `os.Exit` calls
  scattered through the tree.
- **Hints, not walls of text.** The message says what broke; the *hint* says what to
  do about it. Hints are surfaced as their own field, so the message stays short and
  the advice stays visible.
- **Quiet when it should be.** Stack traces and details appear only when debug logging
  is on. A user pressing <kbd>Ctrl</kbd>+<kbd>C</kbd> is not a failure, so
  `LevelFatalQuiet` exits with the right code without shouting at them.
- **Framework-free.** Only `cockroachdb/errors`. No CLI framework, no config system —
  a `depfootprint_test.go` guard keeps it that way, and printing usage goes through a
  caller-supplied seam.

## Where next

<div class="grid cards" markdown>

- :material-rocket-launch: **[Getting started](getting-started.md)** — report your
  first error with a hint and an exit code.
- :material-lightbulb-on-outline: **[Write actionable errors](how-to/actionable-errors.md)**
  — what separates a good error from a bad one.
- :material-exit-run: **[Control the exit code](how-to/exit-codes.md)** — attach a code
  to the error value.
- :material-keyboard-esc: **[Handle interrupts quietly](how-to/handle-interrupts.md)**
  — Ctrl-C is a choice, not a crash.
- :material-lifebuoy: **[Add a support channel](how-to/support-channel.md)** —
  implement `HelpConfig`.
- :material-test-tube: **[Test error handling](how-to/test-error-handling.md)** —
  capture output, inject the exit func, use the mock.

</div>

## Reference

The Go API reference is on
**[pkg.go.dev](https://pkg.go.dev/gitlab.com/phpboyscout/go/errorhandling)**.
