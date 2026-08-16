# Log fields

Everything the handler emits is a single `slog` record: a message, the error as a
structured group, and at most three fields the *process* knows and the error cannot.

The message is `err.Error()` — the fully wrapped message, with no prefix prepended and
no hint appended — unless the error carries an [`Outcome`](api.md#outcome) with a
`Message`, which replaces it.

## Field summary

| Key | Constant | Appears when |
|---|---|---|
| [`err`](#err) | `KeyError` | always |
| [`prefix`](#prefix) | `KeyPrefix` | [`WithPrefix`](api.md#withprefix) was given a non-empty string |
| [`help`](#help) | `KeyHelp` | a `HelpConfig` was supplied and returned a non-empty string |
| [`stacktrace`](#stacktrace) | `KeyStacktrace` | the logger reports debug as enabled, and the error carries a stack |

Fields appear in that order: `err` first, then `prefix`, `help`, `stacktrace`.

There is no separate record for anything. One call to `Fatal`, `Error` or `Warn`
produces exactly one record.

## err

The error itself, handed to `slog` rather than taken apart. Anything implementing
`slog.LogValuer` — which every `gitlab.com/phpboyscout/go/errors` value does — arrives as
a **group**:

```
err.msg    = "could not resolve"          the wrapped message
err.kind   = "errors.basic"               the error's kind
err.hint   = ["check the token"]          every hint attached
err.detail = ["dial tcp: no such host"]   every detail attached
err.host   = "codeberg.org"               any attribute from errors.WithAttrs
```

`kind` is what a query filters on. It is why this module no longer emits a second,
fixed line to say "this was an assertion failure": the record already says so, in a
field, without a string prefix anyone has to match.

Two things worth knowing:

- **Hints and details both appear at every level**, debug or not. Details are not
  debug-gated — if something must not reach an ordinary user, do not attach it.
- **A plain error is not a group.** An `errors.New` from the standard library, or an
  `fmt.Errorf`, has no `LogValue`, so the field is just `err="plain stdlib"` with no
  kind, no hint and no stack.

## prefix

The string given to [`WithPrefix`](api.md#withprefix), attached as a structured
attribute and **not** prepended to the message. A report reads
`msg="cache write failed" prefix=cache-update` rather than folding the two together, so
the message stays exactly what the error says — greppable, and safe for anything
matching on it.

Omitted when the prefix is empty. Use [`Prefix`](api.md#prefix) to assemble one from
parts.

## help

The string returned by the [`HelpConfig`](api.md#helpconfig) passed to
[`New`](api.md#new).

Omitted when no `HelpConfig` was supplied, and omitted when `SupportMessage()` returns
an empty string — that is the designed way to stay silent when no channel is
configured. There is no other suppression: it is attached to every report, at every
level, regardless of debug, including reports of this module's own sentinels.

## stacktrace

The rendered stack, kept out of the [`err`](#err) group deliberately — a stack is large
and rarely what a log line is for, so it is attached separately and only where somebody
has asked for everything.

**Present only when the logger reports debug as enabled** — the handler asks
`Logger.Enabled(ctx, slog.LevelDebug)` rather than reading a level from configuration,
so whatever your `slog` handler considers debug is what governs.

How many frames are rendered is bounded by [`DefaultStackDepth`](api.md#defaultstackdepth)
and adjustable per report with [`WithStackDepth`](api.md#withstackdepth). That bounds
*reporting*, not capture: `go/errors` captures more than this, and
`errors.StackOf(err)` still reaches all of it.

An error carrying no stack — anything from outside `go/errors` — produces no field at
all rather than an empty one.

## What the handler never emits

- **A timestamp, a source location, or a level name.** Those are the `slog` handler's
  job, and the handler you construct decides whether they appear.
- **Anything redacted.** Messages, hints, details and attributes are passed through
  verbatim. Do not attach a credential, a token or personal data to an error and expect
  the report to hide it.
