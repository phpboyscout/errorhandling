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
on. It wraps [cockroachdb/errors](https://github.com/cockroachdb/errors) with a small
reporting pipeline: attach a **hint** telling the user what to do, attach an **exit
code** to the error itself, route by level, and surface stack traces only when debug
logging is on.

## Design

- **Framework-free.** The only dependency is `cockroachdb/errors`. No CLI framework,
  no config system, no logging library — the logging seam is a plain `*slog.Logger`
  and a `depfootprint_test.go` guard enforces the boundary.
- **The error carries the exit code.** `WithExitCode(err, 3)` travels with the error,
  so the code that knows *why* something failed decides how the process exits, and
  `main` stays a one-liner.
- **Hints are for humans.** A wrapped error says what broke; a hint says what to do
  about it. Both are rendered, with the hint kept out of the machine-facing message.
- **Quiet on purpose.** Stack traces and details appear only when the logger has debug
  enabled; `LevelFatalQuiet` exits with the attached code while logging at debug, for
  expected terminations such as a SIGINT where an error line would be noise.
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
	"log/slog"
	"os"

	"github.com/cockroachdb/errors"

	"gitlab.com/phpboyscout/go/errorhandling"
)

func main() {
	handler := errorhandling.New(slog.Default(), nil)

	if err := run(); err != nil {
		handler.Fatal(err) // logs message + hint, exits with the attached code
	}
}

func run() error {
	err := errors.New("config file not found")

	return errorhandling.WithExitCode(
		errorhandling.WithUserHint(err, "Run 'mytool init' to create one"),
		3,
	)
}
```

## What's inside

- **Reporting** — `ErrorHandler` (`Check` / `Fatal` / `Error` / `Warn`), `New`, and the
  `WithExitFunc` option.
- **Hints** — `WithUserHint`, `WithUserHintf`, `WrapWithHint`.
- **Exit codes** — `WithExitCode`, `ExitCode`, `ExitCodeUsage`.
- **Levels** — `LevelFatal`, `LevelFatalQuiet`, `LevelError`, `LevelWarn`.
- **Sentinels** — `ErrNotImplemented` (with `NewErrNotImplemented` for an issue link)
  and `ErrRunSubCommand`, which triggers the usage printer.
- **Help channels** — the `HelpConfig` interface; you supply the implementation.
- **`mocks`** — published testify mocks of `ErrorHandler` and `HelpConfig`.

## What it does not do

No signal handling, no panic recovery, no redaction, no crash reporting, no output
writer of its own — reports go to the `*slog.Logger` you supply. Inside reporting, the
sentinels discard anything attached to them, `errors.Join` loses hints, and an exit code
does not survive being encoded across a process boundary. All of it is listed under
[Limitations](https://errorhandling.go.phpboyscout.uk/reference/limitations/).

## Documentation

Full guides and the reporting model: **[errorhandling.go.phpboyscout.uk](https://errorhandling.go.phpboyscout.uk)**.
Reference — every symbol, level, exit code and log field:
**[/reference](https://errorhandling.go.phpboyscout.uk/reference/)**.
Signatures and doc comments: **[pkg.go.dev](https://pkg.go.dev/gitlab.com/phpboyscout/go/errorhandling)**.

## License

See [LICENSE](LICENSE).
