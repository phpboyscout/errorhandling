# Getting started

This tutorial builds a tiny CLI that fails on purpose, and improves the failure step by
step: first a bare error, then one with a hint, then one that controls its own exit
code. By the end you'll have seen the whole shape of the package.

## Prerequisites

- Go 1.26 or newer.

## 1. Create a module

```bash
mkdir errdemo && cd errdemo
go mod init errdemo
go get gitlab.com/phpboyscout/go/errorhandling github.com/cockroachdb/errors
```

## 2. Report an error

Create a handler and hand it a failure. `New` takes a `*slog.Logger` and an optional
[support channel](how-to/support-channel.md) (`nil` here):

```go
package main

import (
	"log/slog"
	"os"

	"github.com/cockroachdb/errors"

	"gitlab.com/phpboyscout/go/errorhandling"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	handler := errorhandling.New(logger, nil)

	if err := run(); err != nil {
		handler.Fatal(err)
	}
}

func run() error {
	return errors.New("config file not found")
}
```

```bash
go run .
# level=ERROR msg="config file not found"
# exit status 1
```

Correct, but unkind: the user knows *what* happened and nothing about what to do.

## 3. Add a hint

The message says what broke; a **hint** says what to do about it. Keep them separate —
the message stays short, the advice stays visible:

```go
func run() error {
	return errorhandling.WithUserHint(
		errors.New("config file not found"),
		"Run 'errdemo init' to create one",
	)
}
```

```bash
go run .
# level=ERROR msg="config file not found" hints="Run 'errdemo init' to create one"
```

Hints are always shown when present — unlike stack traces, they are not debug-gated,
because advice is useless if the user can't see it.

## 4. Control the exit code

Scripts branch on exit codes. The code that *knows why* the failure happened should
choose it, so attach it to the error and let it travel up:

```go
func run() error {
	err := errors.New("config file not found")
	err = errorhandling.WithUserHint(err, "Run 'errdemo init' to create one")

	return errorhandling.WithExitCode(err, 3)
}
```

```bash
go run . ; echo "exit=$?"
# level=ERROR msg="config file not found" hints="Run 'errdemo init' to create one"
# exit=3
```

Nothing between `run` and `main` had to know about the code — `WithExitCode` is
transparent to `errors.Is`/`errors.As` and survives further wrapping.

## 5. Add context as it bubbles up

Wrap as the error passes through layers. Each wrap adds context and a stack frame:

```go
func run() error {
	if err := loadConfig(); err != nil {
		return errors.Wrap(err, "startup failed")
	}

	return nil
}

func loadConfig() error {
	err := errors.New("config file not found")
	err = errorhandling.WithUserHint(err, "Run 'errdemo init' to create one")

	return errorhandling.WithExitCode(err, 3)
}
```

```
level=ERROR msg="startup failed: config file not found" hints="Run 'errdemo init' to create one"
```

The hint and exit code survived the wrap.

## 6. See the stack trace

Stack traces are captured automatically but shown only when the logger has **debug**
enabled — so normal output stays clean and diagnostics are one flag away:

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}))
```

Re-run and the report gains a `stacktrace` field showing where the error was created
and each point it was wrapped.

## Where next

- **[Write actionable errors](how-to/actionable-errors.md)** — what makes a hint good.
- **[Control the exit code](how-to/exit-codes.md)** — conventions and the fatal path.
- **[Handle interrupts quietly](how-to/handle-interrupts.md)** — Ctrl-C shouldn't look
  like a crash.
- **[The reporting model](explanation/reporting-model.md)** — why reporting happens at
  the top, and what's rendered when.
