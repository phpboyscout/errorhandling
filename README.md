<div align="center">

# errorhandling

**Structured, user-friendly error reporting for Go CLIs — actionable hints, exit codes carried on the error, debug-gated stack traces, and a pluggable support-channel message**

[![Go Reference](https://pkg.go.dev/badge/gitlab.com/phpboyscout/go/errorhandling.svg)](https://pkg.go.dev/gitlab.com/phpboyscout/go/errorhandling)
[![Pipeline](https://gitlab.com/phpboyscout/go/errorhandling/badges/main/pipeline.svg)](https://gitlab.com/phpboyscout/go/errorhandling/-/pipelines)
[![Coverage](https://gitlab.com/phpboyscout/go/errorhandling/badges/main/coverage.svg)](https://gitlab.com/phpboyscout/go/errorhandling/-/graphs/main/charts)
[![phpboyscout Go toolkit](https://img.shields.io/badge/phpboyscout-Go%20toolkit-554488?logo=gitlab&logoColor=white)](https://go.phpboyscout.uk)

<em>Part of the <a href="https://go.phpboyscout.uk">phpboyscout Go toolkit</a> &mdash; small, framework-free Go modules extracted from <a href="https://gitlab.com/phpboyscout/go-tool-base">go-tool-base</a>. Docs: <a href="https://errorhandling.go.phpboyscout.uk">errorhandling.go.phpboyscout.uk</a></em>

</div>

---

`gitlab.com/phpboyscout/go/errorhandling` turns an error into output a user can act
on. It adds a small reporting pipeline over
[go/errors](https://errors.go.phpboyscout.uk): attach a **hint** telling the user what to
do, attach an **exit code** or an **outcome** to the error itself, and surface stack
traces only when debug logging is on.

## Design

- **Framework-free, and dependency-free.** The only dependency is `go/errors`, which is
  stdlib-only. No CLI framework, no config system, no logging library — the logging seam
  is a plain `*slog.Logger` and a `depfootprint_test.go` guard enforces the boundary.
- **The error carries the exit code.** `WithExitCode(err, 3)` travels with the error,
  so the code that knows *why* something failed decides how the process exits, and
  `main` stays a one-liner.
- **Hints are for humans.** A wrapped error says what broke; a hint says what to do
  about it. Both are rendered, with the hint kept out of the machine-facing message.
- **The error carries how the program should end.** An `Outcome` states the exit code,
  the log level, and whether to print usage — so an error can be terminal *and*
  successful, which a closed switch over sentinels could never express.
- **Nothing exits.** `Fatal` reports and *returns* the code it believes the process
  should use; `main` owns termination, so no deferred cleanup is skipped by a library.
- **Quiet on purpose.** Stack traces appear only when the logger has debug enabled, and
  `Quietly()` demotes a fatal report to debug for expected terminations such as a SIGINT
  where an error line would be noise.
- **Bring your own usage printer.** Printing usage for a parent command goes through
  the `SetUsage(func() error)` seam — with Cobra that is `SetUsage(cmd.Usage)` — so
  this module never imports a CLI framework.

## Install

```bash
go get gitlab.com/phpboyscout/go/errorhandling
```

## Quick start

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"gitlab.com/phpboyscout/go/errorhandling"
	"gitlab.com/phpboyscout/go/errors"
)

func main() {
	handler := errorhandling.New(slog.Default(), nil)

	if err := run(); err != nil {
		// Logs the message, kind and hint as one structured record, and hands
		// main the code to exit on.
		os.Exit(handler.Fatal(context.Background(), err))
	}
}

func run() error {
	err := errors.WithHint(
		errors.New("config file not found"),
		"Run 'mytool init' to create one",
	)

	return errorhandling.WithExitCode(err, 3)
}
```

## What's inside

- **Reporting** — `ErrorHandler` (`Fatal` / `Error` / `Warn`) and `New`.
- **Report options** — `WithPrefix`, `Quietly`, `WithStackDepth`.
- **Outcomes** — `Outcome`, `WithOutcome`, `OutcomeOf`.
- **Exit codes** — `WithExitCode`, `ExitCode`, `ExitCodeUsage`.
- **Sentinels** — `ErrNotImplemented` (with `NewErrNotImplemented` for an issue link),
  `ErrRunSubCommand` and `ErrUnknownSubCommand`, which print usage, and
  `ErrAssertionFailure` (with `NewAssertionFailure`).
- **Help channels** — the `HelpConfig` interface; you supply the implementation.
- **`mocks`** — published testify mocks of `ErrorHandler` and `HelpConfig`.

## What it does not do

No process exit, no signal handling, no panic recovery, no redaction, no crash
reporting, no output writer of its own — reports go to the `*slog.Logger` you supply.
Inside reporting, an outcome overrides the level and code you asked for, `Quietly()` is
ignored by `Error` and `Warn`, and details are not debug-gated. All of it is listed
under
[Limitations](https://errorhandling.go.phpboyscout.uk/reference/limitations/).

## Documentation

Full guides and the reporting model: **[errorhandling.go.phpboyscout.uk](https://errorhandling.go.phpboyscout.uk)**.
Reference — every symbol, level, exit code and log field:
**[/reference](https://errorhandling.go.phpboyscout.uk/reference/)**.
Signatures and doc comments: **[pkg.go.dev](https://pkg.go.dev/gitlab.com/phpboyscout/go/errorhandling)**.

## License

See [LICENSE](LICENSE).
