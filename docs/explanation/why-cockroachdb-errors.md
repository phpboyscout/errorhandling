# Why cockroachdb/errors

This module is built on [`cockroachdb/errors`](https://github.com/cockroachdb/errors)
rather than the standard library or one of the older wrapper packages. That choice
shapes the whole design, so it's worth stating plainly — including what this module is
*not*.

## This is a reporting layer, not an abstraction

**The module does not wrap or hide `cockroachdb/errors`.** You import it directly and
use its API. The helpers here — `WithUserHint`, `WrapWithHint`, `NewAssertionFailure` —
are thin conveniences that exist for **discoverability and convention**, not
encapsulation. Using `errors.WithHint` directly instead of `WithUserHint` is equally
correct.

That was a deliberate call: an abstraction layer over a stable, drop-in library adds
indirection without buying anything. What this module adds is the *reporting* half —
levels, exit codes, debug gating, help channels — which the errors library has no
opinion about.

## What the error library must provide

The requirements came from real gaps in a previous implementation:

**Hints as first-class data.** Without them, the only way to give a user guidance is to
embed it in the message — which **mixes user-facing content with programmatic error
identity**. The message becomes long, unstable, and unsafe to match on. A separate hint
channel is what makes [actionable errors](../how-to/actionable-errors.md) possible at
all.

**Structured metadata.** Attaching machine-readable context — issue links, telemetry
keys, safe (PII-free) details — without touching the message string. `ErrNotImplemented`
carrying a tracking URL is exactly this.

**Stack traces without ceremony.** Captured automatically at creation and every wrap,
rendered with plain `fmt.Sprintf("%+v", err)` — no type assertion, no bespoke
`.ErrorStack()` call.

**Assertion-failure markers.** A way to distinguish a *programming* error (an invariant
was violated — nobody outside your team can act on it) from an *operational* one (the
world is in a state the user can fix). Different audiences, different reporting.

**`Is`/`As` that work through every layer**, including across a network boundary, so
sentinel identity survives arbitrary wrapping — which is what lets this module attach
an exit code to an error without breaking anything downstream.

**Multi-error support** for concurrent work, and a wrapping API without a vestigial
skip-depth parameter.

## The cost, and why it's acceptable

`cockroachdb/errors` pulls in `redact`, `logtags`, protobuf, and `getsentry/sentry-go`.
That's a real dependency footprint for an error package.

It's accepted because the library is well-maintained and widely used in production, and
because **the Sentry dependency is inert unless you explicitly configure it** — it's
present for the optional `BuildSentryReport` path, not something the module activates.

## Coming from another errors package

If you're migrating, the mapping is mechanical:

| Was | Now |
|---|---|
| `fmt.Errorf("...: %w", err)` | `errors.Wrap(err, "...")` |
| `errors.Wrap(err, 0)` *(go-errors)* | `errors.WithStack(err)` |
| `errors.WrapPrefix(err, "msg", 0)` | `errors.Wrap(err, "msg")` |
| `err.(*errors.Error).ErrorStack()` | `fmt.Sprintf("%+v", err)` |
| `errors.Errorf(...)` | `errors.Newf(...)` |

The one thing that genuinely breaks is code type-asserting to another library's
concrete error type (`var e *goerrors.Error; errors.As(err, &e)`). Sentinel comparisons,
wrapping, and `errors.Is`/`As` all carry over unchanged.

Plain stdlib errors still work everywhere — the handler reports a `fmt.Errorf` error
perfectly well. You simply get no stack trace for it, which is the reason to prefer
`errors.Newf`.

## What this unlocks later

Capabilities the foundation makes available, whether or not you use them today: Sentry
reports via `errors.BuildSentryReport()`, error domains, telemetry keys, PII-safe
reporting through `errors.Safe()`/`GetSafeDetails()`, and network-portable error types
via the protobuf encoding.

## Related

- [The reporting model](reporting-model.md)
- [Write actionable errors](../how-to/actionable-errors.md)
