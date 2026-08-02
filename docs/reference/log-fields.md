# Log fields

Everything the handler emits is a `slog` record: a message plus structured fields. This
is the complete set of field keys it can produce, what each contains, and the condition
that has to hold for it to appear at all.

The message itself is always `err.Error()` — the fully wrapped message, with no prefix
prepended and no hint appended.

## Field summary

| Key | Constant | Appears when |
|---|---|---|
| [`prefix`](#prefix) | — | A non-empty prefix was passed and the error is not a special error |
| [`stacktrace`](#stacktrace) | `KeyStacktrace` | The logger has debug enabled |
| [`hints`](#hints) | `KeyHints` | The error carries at least one hint |
| [`details`](#details) | `KeyDetails` | The logger has debug enabled *and* the error carries details |
| [`help`](#help) | `KeyHelp` | A `HelpConfig` was supplied and returned a non-empty string |
| [`url`](#url) | — | Only on the `Track progress` line for an unimplemented error with an issue link |
| [`error`](#error) | — | Only on the `Internal error (assertion failure)` line |

Fields appear in that order within a record: `prefix` first, then `stacktrace`,
`hints`, `details`, `help`.

## prefix

The `prefix` argument to [`Check`](api.md#check), or the concatenation of the variadic
prefixes given to `Fatal`/`Error`/`Warn`.

It is attached as a structured attribute, **not** prepended to the message. So a report
reads `msg="cache write failed" prefix=cache-update` rather than folding the two
together. The message stays exactly what the error says, which keeps it greppable and
safe for anything matching on it.

Omitted when the prefix is empty, and omitted entirely on the
[special-error path](levels.md#what-a-special-error-does-at-every-level).

## stacktrace

The `%+v` rendering of the error with the leading copy of the message stripped, so the
field holds only the cause chain and stack frames. Two normalisations are applied so it
renders cleanly inside log formatters that wrap multi-line values: cockroachdb's
`"  | "` line prefix becomes four spaces, and tab characters become two spaces.

**Present only when the logger reports debug as enabled** — the handler asks
`Logger.Enabled(ctx, slog.LevelDebug)` rather than reading a level from configuration,
so whatever your `slog` handler considers debug is what governs.

When the error carries no extra information — a plain `errors.New` from the standard
library, or an `fmt.Errorf` — the field reads `(no stack trace captured)`. That string
is a deliberate signal: it distinguishes "no trace exists" from a trace that happens to
look like the message.

## hints

Every hint attached to the error, flattened by `errors.FlattenHints` into one string.
Where an error carries more than one hint they are joined with a `--` separator on its
own line, so `hints="first\n--\nsecond"`.

**Present at every level, debug or not.** A hint is the actionable half of an error and
is useless if the user has to turn on debug logging to see it.

Two cases produce a hint you did not write:

- An [assertion failure](levels.md#what-an-assertion-failure-does) carries
  `cockroachdb/errors`' own generic "you have encountered an unexpected error" text.
- Nothing else. If the field is unexpectedly empty, the likely cause is
  [an `errors.Join`](limitations.md#hints-are-lost-when-errors-are-joined), which drops
  hints entirely.

## details

Details attached with `errors.WithDetail`, flattened by `errors.FlattenDetails` and
joined with the same `--` separator that [`hints`](#hints) uses.

**Present only when debug is enabled**, which is the whole reason details exist as a
separate channel from hints: a raw HTTP body or a driver-level code belongs here, where
a user diagnosing a problem can ask for it and an ordinary user never sees it.

The same information also appears inside [`stacktrace`](#stacktrace) on the same
record, because `%+v` includes it. That duplication is expected.

## help

The string returned by the [`HelpConfig`](api.md#helpconfig) passed to
[`New`](api.md#new).

Omitted when no `HelpConfig` was supplied, and omitted when `SupportMessage()` returns
an empty string. There is no other way to suppress it — it is attached to every report
on the normal path, at every level, regardless of debug.

It is **not** attached on the
[special-error path](levels.md#what-a-special-error-does-at-every-level), so a
"subcommand required" warning never carries the support message.

## url

Emitted on a separate `INFO` record with the message `Track progress`, carrying the
issue-tracker link given to [`NewErrNotImplemented`](api.md#newerrnotimplemented).

Only the **first** attached issue link is used. The value is whatever string was passed
in — it is not validated, so an empty string produces `url=""`.

## error

Emitted on the `ERROR Internal error (assertion failure)` record, carrying the error
*value* rather than its message.

How much that prints is decided by your `slog` handler, not by this package:
`TextHandler` formats an arbitrary value with `%+v` and so prints the full stack;
`JSONHandler` special-cases errors and prints the message alone. See
[what an assertion failure does](levels.md#what-an-assertion-failure-does).

## What the handler never emits

- **A field naming the level string.** The level chooses the `slog` level and nothing
  else; `"fatal-quiet"` never appears in the output.
- **A timestamp, a source location, or a level name.** Those are the `slog` handler's
  job, and the handler you construct decides whether they appear.
- **Anything redacted.** Messages, hints and details are passed through verbatim.
  `cockroachdb/errors` can carry PII-safe details, but this package does not use that
  distinction — do not attach a secret to an error and expect the report to hide it.
