# Getting started

Build a tiny CLI that fails on purpose, then improve the failure step by step: first a
bare error, then one with a hint, then one that controls its own exit code. By the end
you'll have seen the whole shape of the package and have something you can copy into a
real tool.

Allow about fifteen minutes. Everything happens in a scratch directory you can delete
afterwards; nothing is installed globally.

## Prerequisites

- Go 1.26 or newer. The module's `go.mod` declares `go 1.26.5`, so an older toolchain
  will refuse to build it.
- A terminal. No editor plugin, no Docker, nothing else.

## 1. Create a module

```bash
mkdir errdemo && cd errdemo
go mod init errdemo
go get gitlab.com/phpboyscout/go/errorhandling github.com/cockroachdb/errors
```

You pull in `cockroachdb/errors` directly, alongside this module. That's intentional —
this package is a reporting layer rather than a wrapper, so you'll use both APIs side
by side.

## 2. Report an error

Create a handler and hand it a failure. `New` takes a `*slog.Logger` and an optional
[support channel](../how-to/support-channel.md) (`nil` here):

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
```

```
time=2026-01-01T00:00:00.000Z level=ERROR msg="config file not found"
exit status 1
```

Correct, but unkind: the user knows *what* happened and nothing about what to do.

Timestamps are trimmed from the output shown from here on. Yours will carry a real one,
because `slog`'s text handler always emits it.

## 3. Add a hint so the user knows what to do

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
```

```
level=ERROR msg="config file not found" hints="Run 'errdemo init' to create one"
```

Hints are always shown when present — unlike stack traces, they are not debug-gated,
because advice is useless if the user can't see it.

## 4. Control the exit code

Scripts branch on exit codes. The code that *knows why* the failure happened should
choose it, so attach it to the error and let it travel:

```go
func run() error {
	err := errors.New("config file not found")
	err = errorhandling.WithUserHint(err, "Run 'errdemo init' to create one")

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
level=ERROR msg="config file not found" hints="Run 'errdemo init' to create one"
exit=3
```

Nothing between `run` and `main` had to know about the code — `WithExitCode` is
transparent to `errors.Is`/`errors.As` and survives further wrapping.

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
	err := errors.New("config file not found")
	err = errorhandling.WithUserHint(err, "Run 'errdemo init' to create one")

	return errorhandling.WithExitCode(err, 3)
}
```

```
level=ERROR msg="startup failed: config file not found" hints="Run 'errdemo init' to create one"
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
level=ERROR msg="startup failed: config file not found" stacktrace="(1) attached stack trace
  -- stack trace:
    main.run
      /home/you/errdemo/main.go:25
    [...repeated from below...]
Wraps: (2) startup failed
Wraps: (3)
Wraps: (4) Run 'errdemo init' to create one
Wraps: (5) attached stack trace
  ..." hints="Run 'errdemo init' to create one"
```

It is long, and it is meant to be — this is the debug view. Notice that the hint and
the exit-code wrapper appear as layers in the chain, which is a quick way to check that
something you attached really made it through.

## Where next

- **[Write actionable errors](../how-to/actionable-errors.md)** — what makes a hint good.
- **[Control the exit code](../how-to/exit-codes.md)** — conventions and the fatal path.
- **[Handle interrupts quietly](../how-to/handle-interrupts.md)** — Ctrl-C shouldn't look
  like a crash.
- **[Report levels](../reference/levels.md)** — exactly what each level logs, and
  whether it exits.
- **[Limitations](../reference/limitations.md)** — what the package deliberately does
  not do, worth reading before you build on it.
- **[The reporting model](../explanation/reporting-model.md)** — why reporting happens at
  the top, and what's rendered when.
