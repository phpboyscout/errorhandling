# errorhandling

**Turn an error into something a user can act on.** Attach a hint that says what to
*do*; attach the exit code to the error itself; report it once, at the top; and keep
stack traces for when someone asks for them.

```bash
go get gitlab.com/phpboyscout/go/errorhandling
```

`gitlab.com/phpboyscout/go/errorhandling` is the error-reporting layer extracted from
[go-tool-base](https://gitlab.com/phpboyscout/go-tool-base). It adds a small reporting
pipeline over [go/errors](https://errors.go.phpboyscout.uk): levels, outcomes, exit
codes, debug-gated stacks, and an optional support-channel message.

## Why

- **Two jobs, kept separate.** *Creating* an error (with context, hints, and a stack)
  is `go/errors`' job. *Reporting* one — deciding the level, the exit code, and what the
  user sees — is this package's. See
  [the reporting model](explanation/reporting-model.md).
- **The error carries its exit code.** `WithExitCode(err, 3)` travels with the error
  through every wrap, so the code that knows *why* something failed decides how the
  process exits — and every exit stays on one path instead of `os.Exit` calls
  scattered through the tree.
- **Hints, not walls of text.** The message says what broke; the *hint* says what to
  do about it. Hints are surfaced as their own field, so the message stays short and
  the advice stays visible.
- **The error carries how the program should end.** An `Outcome` states the exit code,
  the level, and whether to print usage — so an error can be terminal *and* successful,
  which a closed switch over sentinels could never express.
- **Quiet when it should be.** Stack traces appear only when debug logging is on. A user
  pressing <kbd>Ctrl</kbd>+<kbd>C</kbd> is not a failure, so `Quietly()` reports at the
  right code without shouting at them.
- **Framework-free, and dependency-free.** `go/errors` is stdlib-only, so adding this
  module adds this module. No CLI framework, no config system — a
  `depfootprint_test.go` guard keeps it that way, and printing usage goes through a
  caller-supplied seam.

## Where next

<div class="grid cards" markdown>

- :material-rocket-launch: **[Getting started](tutorials/getting-started.md)** — report
  your first error with a hint and an exit code.
- :material-lightbulb-on-outline: **[Write actionable errors](how-to/actionable-errors.md)**
  — what separates a good error from a bad one.
- :material-exit-run: **[Control the exit code](how-to/exit-codes.md)** — attach a code
  to the error value.
- :material-keyboard-esc: **[Handle interrupts quietly](how-to/handle-interrupts.md)**
  — Ctrl-C is a choice, not a crash.
- :material-lifebuoy: **[Add a support channel](how-to/support-channel.md)** —
  implement `HelpConfig`.
- :material-test-tube: **[Test error handling](how-to/test-error-handling.md)** —
  capture output, assert on the returned code, use the mock.
- :material-book-open-variant: **[Reference](reference/index.md)** — every symbol,
  level, exit code and log field, and what each does with bad input.
- :material-block-helper: **[Limitations](reference/limitations.md)** — what the
  package deliberately does not do.

</div>

## What this package does not do

It reports errors. It does not exit, catch signals, recover panics, redact secrets,
retry anything, or send a crash report — `Fatal` returns the code it thinks the process
should use and leaves acting on it to `main`. Reports go to the `*slog.Logger` you
supply and nowhere else — there is no writer, no colour control and no format control
of its own.

Within reporting, the sharp edges worth knowing before you commit: an error carrying an
outcome [overrides the level and code you asked for](reference/limitations.md#an-outcome-overrides-the-level-and-code-the-caller-asked-for),
[`Quietly` is ignored by `Error` and `Warn`](reference/limitations.md#quietly-is-ignored-by-error-and-warn),
and [details are not debug-gated](reference/limitations.md#details-are-not-debug-gated).
The full list is on [the limitations page](reference/limitations.md).

## Reference

The [reference section](reference/index.md) covers every exported symbol, the exact
behaviour of each level, how the exit code is resolved, and the fields a report emits.

Signatures and doc comments are also published on
**[pkg.go.dev](https://pkg.go.dev/gitlab.com/phpboyscout/go/errorhandling)**.

## Further reading

The blog carries a curated route through this subject: **[Building a command-line tool in Go](https://phpboyscout.uk/topics/building-a-cli-in-go/)** collects
everything written about it, ordered so you can start at the beginning rather
than newest-first.

!!! tip "Ask phpbotscout"

    ![phpbotscout](https://phpboyscout.uk/images/projects/logo-phpbotscout.png){ width="84" align=left style="border-radius:10px;margin-right:1rem" }

    He answers questions about the projects over on the Discord, citing the docs
    where they already cover it, and offering to raise an issue where they don't.
    Bring a bug, an idea, or a questionable engineering decision.

    [Join the Discord](https://discord.gg/mQzGbmGyzZ){ .md-button .md-button--primary }
