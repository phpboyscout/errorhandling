# Print usage for a parent command

A command that only groups subcommands — `mytool config`, with `get`/`set`/`list`
beneath it — has nothing to do when invoked alone. The useful response is to show the
user what they *could* have typed.

Return `ErrRunSubCommand` and the handler prints usage and warns:

```go
func runConfig(ctx context.Context) error {
	return errorhandling.ErrRunSubCommand
}
```

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

## Related

- [The reporting model](../explanation/reporting-model.md#sentinels-that-mean-something)
- [Write actionable errors](actionable-errors.md)
