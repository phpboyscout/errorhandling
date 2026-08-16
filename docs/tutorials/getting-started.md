# Getting started

Build a tiny CLI that fails on purpose, then improve the failure step by step: first a
bare error, then one with a hint, then one that controls its own exit code. By the end
you'll have seen the whole shape of the package and have something you can copy into a
real tool.

Allow about fifteen minutes. Everything happens in a scratch directory you can delete
afterwards; nothing is installed globally.

## Prerequisites

- Go 1.26 or newer. The module's `go.mod` declares `go 1.26.6`, so an older toolchain
  will refuse to build it.
- A terminal. No editor plugin, no Docker, nothing else.

## 1. Create a module

```bash
mkdir errdemo && cd errdemo
go mod init errdemo
go get gitlab.com/phpboyscout/go/errorhandling gitlab.com/phpboyscout/go/errors
```

You pull in `go/errors` directly, alongside this module. That's intentional — this
package is a reporting layer rather than a wrapper, so you'll use both APIs side by
side. `go/errors` has no dependencies of its own.

## 2. Report an error

Create a handler and hand it a failure. `New` takes a `*slog.Logger` and an optional
[support channel](../how-to/support-channel.md) (`nil` here):

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
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	handler := errorhandling.New(logger, nil)

	if err := run(); err != nil {
		os.Exit(handler.Fatal(context.Background(), err))
	}
}

func run() error {
	return errors.New("config file not found")
}
```

```bash
go run .
```

```
time=2026-01-01T00:00:00.000Z level=ERROR msg="config file not found" err.msg="config file not found" err.kind=errors.basic
exit status 1
```

Two things to notice.

**`Fatal` does not exit.** It reports, and returns the code it believes the process
should use — `os.Exit` in `main` is what acts on it. That is why this module can be
called from anywhere without swallowing your deferred cleanup.

**The error arrives as a group**, not a flattened string: `err.msg` and `err.kind` are
separate fields a log query can filter on. Everything you attach from here on joins it.

Correct, but unkind: the user knows *what* happened and nothing about what to do.

Timestamps are trimmed from the output shown from here on. Yours will carry a real one,
because `slog`'s text handler always emits it.

## 3. Add a hint so the user knows what to do

The message says what broke; a **hint** says what to do about it. Keep them separate —
the message stays short, the advice stays visible:

```go
func run() error {
	return errors.WithHint(
		errors.New("config file not found"),
		"Run 'errdemo init' to create one",
	)
}
```

```bash
go run .
```

```
level=ERROR msg="config file not found" err.msg="config file not found" err.kind=errors.basic err.hint="[Run 'errdemo init' to create one]"
```

Hints are always shown when present — unlike stack traces, they are not debug-gated,
because advice is useless if the user can't see it.

## 4. Control the exit code

Scripts branch on exit codes. The code that *knows why* the failure happened should
choose it, so attach it to the error and let it travel:

```go
func run() error {
	err := errors.WithHint(
		errors.New("config file not found"),
		"Run 'errdemo init' to create one",
	)

	return errorhandling.WithExitCode(err, 3)
}
```

To see the code you have to build the binary and run that. **`go run` will not show you
this**: it exits `1` whatever your program returned, and prints the real status as a
separate `exit status 3` line on stderr. That catches everyone out once, so switch to a
build here:

```bash
go build -o errdemo . && ./errdemo ; echo "exit=$?"
```

```
level=ERROR msg="config file not found" err.msg="config file not found" err.kind=errorhandling.exit_code err.hint="[Run 'errdemo init' to create one]"
exit=3
```

Nothing between `run` and `main` had to know about the code — `WithExitCode` is
transparent to `errors.Is`/`errors.As` and survives further wrapping. The hint is still
there, and `err.kind` now names the outermost attachment.

## 5. Add context as the error bubbles up

Wrap as the error passes through layers. Each wrap adds context and a stack frame:

```go
func run() error {
	if err := loadConfig(); err != nil {
		return errors.Wrap(err, "startup failed")
	}

	return nil
}

func loadConfig() error {
	err := errors.WithHint(
		errors.New("config file not found"),
		"Run 'errdemo init' to create one",
	)

	return errorhandling.WithExitCode(err, 3)
}
```

```
level=ERROR msg="startup failed: config file not found" err.msg="startup failed: config file not found" err.kind=errorhandling.exit_code err.hint="[Run 'errdemo init' to create one]"
```

The hint and the exit code both survived the wrap — rebuild, check `$?`, and it is
still `3`.

## 6. See the stack trace

Stack traces are captured automatically but shown only when the logger has **debug**
enabled, so normal output stays clean and diagnostics are one setting away:

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}))
```

Rebuild and the report gains a `stacktrace` field showing where the error was created
and each point it was wrapped:

```
level=ERROR msg="startup failed: config file not found" err.msg="startup failed: config file not found" err.kind=errorhandling.exit_code err.hint="[Run 'errdemo init' to create one]" stacktrace="main.loadConfig
\t/home/you/errdemo/main.go:31
main.run
\t/home/you/errdemo/main.go:22
main.main
\t/home/you/errdemo/main.go:16
..."
```

The stack lives outside the `err` group deliberately: it is large, and it is rarely what
a log line is for. It is bounded to
[twenty frames](../reference/api.md#defaultstackdepth) by default, and
[`WithStackDepth`](../reference/api.md#withstackdepth) changes that per report.

## Where next

- **[Write actionable errors](../how-to/actionable-errors.md)** — what makes a hint good.
- **[Control the exit code](../how-to/exit-codes.md)** — conventions and the fatal path.
- **[Handle interrupts quietly](../how-to/handle-interrupts.md)** — Ctrl-C shouldn't look
  like a crash.
- **[Report levels](../reference/levels.md)** — exactly what each method logs, and what
  an outcome does to it.
- **[Limitations](../reference/limitations.md)** — what the package deliberately does
  not do, worth reading before you build on it.
- **[The reporting model](../explanation/reporting-model.md)** — why reporting happens at
  the top, and what's rendered when.
