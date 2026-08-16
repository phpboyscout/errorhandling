# Report levels

Which `slog` level a report is logged at is decided by three things, in this order: the
method you called, the [`Quietly`](api.md#quietly) option, and any
[`Outcome`](api.md#outcome) the error carries. The last one wins.

Whether the process ends is a separate decision, and not this module's:
[`Fatal`](api.md#fatal) *returns* an exit code and leaves acting on it to `main`.

## What each method logs at

| Method | Logs at | Returns a code | Honours `Quietly` |
|---|---|---|---|
| [`Fatal`](api.md#fatal) | `ERROR` | yes | yes |
| [`Error`](api.md#error) | `ERROR` | no | **no** |
| [`Warn`](api.md#warn) | `WARN` | no | **no** |

A nil error reports nothing at any of them. `Fatal(ctx, nil)` returns `0` without
logging, so a fatal path guarded by a nil check needs no second guard.

### Quietly only applies to Fatal

[`Quietly`](api.md#quietly) demotes the line to `DEBUG` without changing the exit code.
It is read on the fatal path only — `Error(ctx, err, Quietly())` still logs at `ERROR`,
silently ignoring the option.

That is deliberate rather than an oversight: the option exists for a termination whose
whole meaning is the exit code, and a non-terminal report has no code to carry the
meaning instead.

## An outcome overrides the level

If the error carries an [`Outcome`](api.md#outcome), its `Level` replaces whatever the
method and `Quietly` decided, and its `Code` replaces the attached exit code:

```go
h.Error(ctx, errorhandling.ErrRunSubCommand)   // logs at WARN, not ERROR
h.Fatal(ctx, errorhandling.ErrRunSubCommand, errorhandling.Quietly())
                                               // logs at WARN, not DEBUG; returns 2
```

The error knows what kind of ending it is; the reporting site usually does not. A caller
that needs a different level has to replace the outcome, not argue with it — see
[Limitations](limitations.md#an-outcome-overrides-the-level-and-code-the-caller-asked-for).

## What the sentinels do

The three sentinels this module ships all carry an outcome, so all three report at
`WARN` whichever method reports them, and all three make `Fatal` return `2`:

| Error | Message | Prints usage | `Fatal` returns |
|---|---|---|---|
| [`ErrRunSubCommand`](api.md#errrunsubcommand) | `subcommand required` | yes | [`2`](exit-codes.md#exitcodeusage) |
| [`ErrUnknownSubCommand`](api.md#errunknownsubcommand) | `unknown subcommand` | yes | [`2`](exit-codes.md#exitcodeusage) |
| [`ErrNotImplemented`](api.md#errnotimplemented) | `command not yet implemented` | no | [`2`](exit-codes.md#exitcodeusage) |

Usage is printed through the [`SetUsage`](api.md#setusage) seam, before the report. With
no printer registered the error is still reported and the code is still `2`; there is
simply no usage output.

**Everything the error carries is still reported.** Wrapping one of these does not lose
the wrap's message, and hints, details, attributes, the prefix, the help message and the
stack at debug all arrive as usual:

```go
errors.Wrap(errorhandling.ErrRunSubCommand, "config")
// WARN config: subcommand required
```

## What an assertion failure does

An error from [`NewAssertionFailure`](api.md#newassertionfailure) is reported like any
other error: **one record**, at the level the method asked for, identified by the kind
`errorhandling.assertion_failure` inside the [`err` group](log-fields.md#err).

```go
h.Warn(ctx, errorhandling.NewAssertionFailure("bad: %s", "x<0"))
// WARN bad: x<0: internal invariant violated   err.kind=errorhandling.assertion_failure
```

It carries no outcome, so it does not override the level and `Fatal` returns `1` unless a
code is attached. The kind is what a query filters on — which is why the second,
fixed `ERROR Internal error (assertion failure)` line earlier versions emitted was
dropped: it said in prose what the record already says in a field.

## Related

- [Exit codes](exit-codes.md) — which code `Fatal` returns
- [Log fields](log-fields.md) — what appears alongside the message
- [The reporting model](../explanation/reporting-model.md) — why the split is drawn here
