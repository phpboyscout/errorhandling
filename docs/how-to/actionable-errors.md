# Write actionable errors

An error that only says *what broke* leaves the user stuck. This guide covers the four
places information can live, and how to choose between them.

## The rule for hints

> A hint tells the user **what to do next** — not what the error was.

The message carries the error's *identity* (and is what `errors.Is` and your logs match
on). The hint carries the *advice*. Keeping them apart is the whole point: advice
embedded in a message makes the message long, unstable, and useless to match on.

```go
// ✅ identity in the message, advice in the hint
return errors.WithHint(
	errors.New("config file not found"),
	"Run 'mytool init' to create one",
)

// ❌ advice welded into the identity
return errors.New("config file not found\n\nSuggestions:\n  - run mytool init\n  - pass --config")
```

Good hints name a **concrete next action** — a command to run, an environment variable
to set, a setting to check:

```go
errors.WithHintf(err, "Set %s or run 'mytool init --token'.", "GITHUB_TOKEN")
errors.WithHint(errors.Wrap(err, "failed to authenticate"),
	"Check that your GITHUB_TOKEN is valid and has the required scopes.")
```

Make the hint discriminate between failure modes. Two failures at the same call site
deserve two different hints:

```go
if err := sql.Open(driver, dsn); err != nil {
	return errors.WithHint(errors.Wrap(err, "failed to connect to database"),
		"Check that the database server is running and the connection string is correct")
}

if err := conn.Ping(); err != nil {
	return errors.WithHint(errors.Wrap(err, "database connection test failed"),
		"The connection was established but the database is not responding — check server health")
}
```

## Four carriers, four audiences

| Carrier | Audience | Shown |
|---|---|---|
| **Message** (`errors.New`, `errors.Wrap`) | matching code, operators | always |
| **Hint** (`errors.WithHint`) | **the end user — how to fix it** | always, at every level |
| **Detail** (`errors.WithDetail`) | developers diagnosing | always, at every level |
| **Attribute** (`errors.WithAttrs`) | log queries and dashboards | always, as its own field |

Plus the **stack trace**, captured automatically and rendered only in debug.

The split that matters is *audience*, not visibility: hints and details both reach the
record. A detail is where a raw HTTP body or a driver-level code goes so it does not
clutter the hint — **not** where a secret goes. Nothing here is redacted or debug-gated,
so anything that must not reach a user must not be attached to the error at all.

## Creating errors: the ladder

1. `errors.New("static message")` — the default.
2. `errors.Newf("invalid port: %d", port)` — when the message needs values.
3. `errors.Wrap(err, "context")` — adding context as an error travels up.
4. `errors.WithStack(err)` — re-returning a sentinel and you only want the stack.
5. `errors.WithHint(err, "…")` — adding advice.
6. `errors.NewSentinel(kind, msg)` — **at package scope**. Plain `New` there captures
   its stack at initialisation, which points at `runtime.doInit` rather than anywhere
   the error was returned.

!!! danger "Never `fmt.Errorf`"
    `fmt.Errorf` captures **no stack trace**, so a failure that reaches your logs has
    no origin. Use `errors.Newf`/`errors.Wrap` from `go/errors` instead —
    same ergonomics, plus a stack. (The handler tolerates plain errors; you just get
    no trace for them.)

## Prefer sentinels, wrap for specifics

Declare the conditions callers need to distinguish as package-level values, then wrap
for the dynamic part. That keeps `errors.Is` working while the message stays specific:

```go
var ErrInvalidPort = errors.NewSentinel(
	"mytool.invalid_port", "invalid port: must be between 1 and 65535")

func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return errors.Wrap(ErrInvalidPort, fmt.Sprintf("port %d", port))
	}

	return nil
}
```

Wrapping preserves identity — `errors.Is(err, ErrInvalidPort)` still matches — so
callers can branch, and the boundary that *recognises* a condition is a natural place
to attach the hint:

```go
if errors.Is(err, ErrTokenExpired) {
	return errors.WithHint(err, "Run 'mytool init --github' to re-authenticate.")
}
```

## Message guidelines

- **Be specific** — name the file path, URL, or config key involved.
- **Be actionable** — if there's a fix, it goes in a hint.
- **Be consistent** — same terminology across your tool.
- **Wrap, don't replace** — add context as the error rises; never discard the cause.

## Flag a bug, not a user error

When a condition means *your code* is wrong rather than the user's input, say so:

```go
if len(items) == 0 {
	return errorhandling.NewAssertionFailure(
		"processItems called with empty slice; this is a bug")
}
```

The message convention is to name the function, state the violated precondition, and say
it is a bug. Don't add a hint: nobody outside your team can act on it, and a hint that
cannot be acted on trains people to ignore hints.

What identifies it in the log is the kind, `errorhandling.assertion_failure`, inside the
[`err` group](../reference/log-fields.md#err) — so a query can count them without
matching on message text. It is otherwise reported like any other error, at whatever
level you asked for.

## Related

- [The reporting model](../explanation/reporting-model.md)
- [Why go/errors](../explanation/why-go-errors.md)
- [Control the exit code](exit-codes.md)
- [Log fields](../reference/log-fields.md) — which of these carriers renders where
