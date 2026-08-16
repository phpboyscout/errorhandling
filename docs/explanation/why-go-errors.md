# Why go/errors

This module is built on [`gitlab.com/phpboyscout/go/errors`](https://errors.go.phpboyscout.uk)
rather than the standard library directly. That choice shapes the whole design, so it is
worth stating plainly — including what this module is *not*.

## This is a reporting layer, not an abstraction

**The module does not wrap or hide `go/errors`.** You import it directly and use its
API. Creation and enrichment — `New`, `Wrap`, `WithHint`, `WithDetail`, `WithAttrs` —
all live there, and this module adds none of its own aliases for them.

What it adds is the *reporting* half: levels, exit codes, outcomes, debug gating and
help channels, which the errors library has no opinion about. An abstraction layer over
a stable library would add indirection without buying anything.

The few constructors here — [`NewErrNotImplemented`](../reference/api.md#newerrnotimplemented),
[`NewAssertionFailure`](../reference/api.md#newassertionfailure) — exist because they
attach *this module's* sentinels, not as conveniences over the errors package.

## What the error library must provide

**Hints and details as first-class data.** Without them, the only way to give a user
guidance is to embed it in the message — which mixes user-facing content with
programmatic error identity, making the message long, unstable and unsafe to match on. A
separate hint channel is what makes
[actionable errors](../how-to/actionable-errors.md) possible at all.

**A kind on every error.** A stable, queryable string identifying what an error *is* —
`errorhandling.assertion_failure`, `errors.basic` — so a log query can filter on identity
rather than on a message prefix. It is what let this module stop emitting a fixed second
line to announce an assertion failure.

**`slog.LogValuer` on the error itself.** The error renders itself as a structured group,
so the handler hands it to `slog` rather than taking it apart. Everything an error
carries reaches the record without this module enumerating it, which is why hints,
details and attributes need no field constants here.

**Stack traces without ceremony**, captured at creation and reachable with
`errors.StackOf` — and, importantly, *not* captured by `NewSentinel`, so a package-level
sentinel does not freeze a stack pointing at `runtime.doInit`.

**`Is`/`As` that work through every layer**, so sentinel identity survives arbitrary
wrapping. That is what lets this module attach an exit code or an outcome to an error
without anything downstream noticing.

## The cost, and why it is acceptable

There is none to speak of. `go/errors` requires nothing outside the standard library, so
adding this module to a binary adds this module.

That is a deliberate reversal. This package was previously built on
`cockroachdb/errors`, which pulled in `redact`, `logtags`, protobuf and
`getsentry/sentry-go` — a real footprint for an error package, accepted at the time
because the Sentry path was inert unless configured. Moving to a stdlib-only foundation
removed the whole graph, and the
[dependency-footprint guard](https://gitlab.com/phpboyscout/go/errorhandling/-/blob/main/depfootprint_test.go)
keeps it from creeping back.

## Coming from cockroachdb/errors

The mapping is mechanical, and every symbol this module used has a same-named
equivalent:

| Was | Now |
|---|---|
| `errors.New`, `errors.Newf`, `errors.Wrap`, `errors.Wrapf` | unchanged |
| `errors.WithHint`, `errors.WithDetail` | unchanged |
| `errors.New` **at package scope** | `errors.NewSentinel(kind, msg)` |
| `errors.AssertionFailedf(...)` | [`NewAssertionFailure(...)`](../reference/api.md#newassertionfailure) |
| `errors.HasAssertionFailure(err)` | `errors.Is(err, ErrAssertionFailure)` |
| `errors.HasUnimplementedError(err)` | `errors.Is(err, ErrNotImplemented)` |
| `errors.FlattenHints(err)` | `errors.Hints(err)` — returns a slice |
| `fmt.Sprintf("%+v", err)` for a stack | `errors.StackOf(err)` |
| `errors.CombineErrors(a, b)` | `errors.Join(a, b)` |

`NewSentinel` at package scope is the one that matters. Plain `New` captures its stack
at package initialisation, so every report of that sentinel points at `runtime.doInit`
rather than anywhere the error was returned.

Plain stdlib errors still work everywhere — the handler reports an `fmt.Errorf` error
perfectly well. It arrives as a bare string rather than a group, with no kind and no
stack, which is the reason to prefer `errors.Newf`.

## Related

- [The reporting model](reporting-model.md)
- [Write actionable errors](../how-to/actionable-errors.md)
