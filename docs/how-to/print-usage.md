# Print usage for a parent command

A command that only groups subcommands — `mytool config`, with `get`/`set`/`list`
beneath it — has nothing to do when invoked alone. The useful response is to show the
user what they *could* have typed.

There are two answers, and which one is right is a decision about your CLI:

- **A bare invocation is a request for help.** Print usage and succeed. Most grouping
  commands work this way, and cobra's `cmd.Usage()` returns `nil`, so this is just
  `return cmd.Usage()`.
- **A bare invocation is a mistake.** Return
  [`ErrRunSubCommand`](../reference/api.md#errrunsubcommand); the handler prints usage,
  warns, and the code is `2`.

```go
func runConfig(ctx context.Context) error {
	return errorhandling.ErrRunSubCommand
}
```

Reach for the sentinel when the command genuinely cannot act on its own and a caller
should be able to tell that from the exit code. The rest of this page is about the seam
that makes either one print.

## Wire up the usage printer

This module has no CLI-framework dependency, so it cannot know how to print your usage.
You supply that through `SetUsage`:

```go
handler.SetUsage(cmd.Usage)   // Cobra: cmd.Usage is already func() error
```

Any `func() error` works — a hand-rolled flag package, a template, or a plain
`fmt.Fprintln`.

## Set it per command, not once globally

The handler stores **one** usage function, so set it to the command that is actually
running. With Cobra, the natural place is the command's pre-run hook, which fires for
the command being executed:

```go
PreRunE: func(cmd *cobra.Command, args []string) error {
	handler.SetUsage(cmd.Usage)

	return nil
},
```

Do this and a failure in `mytool config` prints `config`'s usage, not the root
command's. Set it once at the root instead and every parent command will print the
root's usage — technically correct, practically useless.

## Send usage to a specific writer

`SetUsage` takes a closure, so redirect output by capturing it:

```go
handler.SetUsage(func() error {
	cmd.SetOut(os.Stderr)

	return cmd.Usage()
})
```

## When no printer is set

`ErrRunSubCommand` is still handled — the "subcommand required" warning is logged and
the error is considered dealt with; there is simply no usage output. That makes the
seam optional: a tool with no usage concept can ignore it entirely.

## When the usage printer itself fails

The error your function returns is **discarded**. Nothing is logged about it and the
report is emitted either way, so a printer that fails silently produces a report with no
usage above it. If that matters — a printer writing to a pipe
that has closed, say — handle the error inside the closure.

## Decorating the sentinel works

Wrap it, hint it, attach attributes — all of it reaches the report, because the sentinel
is an ordinary error carrying an [`Outcome`](../reference/api.md#outcome) rather than a
special case the handler switches on:

```go
errors.Wrapf(errorhandling.ErrUnknownSubCommand,
	"unknown command %q for %q", args[0], cmd.CommandPath())

// WARN unknown command "bogus" for "tool alpha": unknown subcommand
```

The one thing you cannot change by attaching to it is the ending: the outcome fixes the
level at warn and the code at `2`, and beats a code attached with `WithExitCode`. To end
differently, attach your own outcome — see
[Limitations](../reference/limitations.md#a-usage-outcomes-exit-code-cannot-be-chosen).

## Reporting a mistyped subcommand

Cobra reports an unknown command for the **root only**, so a parent command that wants
to catch a typo has to do it in its own run function:

```go
RunE: func(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return errors.Wrapf(errorhandling.ErrUnknownSubCommand,
			"unknown command %q for %q", args[0], cmd.CommandPath())
	}

	return cmd.Usage()
},
```

That prints usage through the same seam and returns `2`, while a bare invocation prints
usage and succeeds.

## Related

- [The reporting model](../explanation/reporting-model.md#the-outcome-belongs-to-the-error-too)
- [Write actionable errors](actionable-errors.md)
- [API reference: SetUsage](../reference/api.md#setusage)
