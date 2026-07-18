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
return errorhandling.WithUserHint(
	errors.New("config file not found"),
	"Run 'mytool init' to create one",
)

// ❌ advice welded into the identity
return errors.New("config file not found\n\nSuggestions:\n  - run mytool init\n  - pass --config")
```

Good hints name a **concrete next action** — a command to run, an environment variable
to set, a setting to check:

```go
errorhandling.WithUserHintf(err, "Set %s or run 'mytool init --token'.", "GITHUB_TOKEN")
errorhandling.WrapWithHint(err, "failed to authenticate",
	"Check that your GITHUB_TOKEN is valid and has the required scopes.")
```

Make the hint discriminate between failure modes. Two failures at the same call site
deserve two different hints:

```go
if err := sql.Open(driver, dsn); err != nil {
	return errorhandling.WrapWithHint(err, "failed to connect to database",
		"Check that the database server is running and the connection string is correct")
}

if err := conn.Ping(); err != nil {
	return errorhandling.WrapWithHint(err, "database connection test failed",
		"The connection was established but the database is not responding — check server health")
}
```

## Four carriers, four audiences

| Carrier | Audience | Shown |
|---|---|---|
| **Message** (`New`, `Wrap`) | matching code, operators | always |
| **Hint** (`WithUserHint`, `errors.WithHint`) | **the end user — how to fix it** | **always, at every level** |
| **Detail** (`errors.WithDetail`) | developers diagnosing | **debug only** |
| **Safe detail** (`errors.WithSafeDetails`) | telemetry / crash reports, PII-free | never in normal output |

Plus the **stack trace**, captured automatically and rendered only in debug.

The split that matters most: **hints are always visible, details are not.** Anything
that would confuse an end user — a raw HTTP body, a driver-level code — belongs in a
detail, never a hint. Anything the user must act on belongs in a hint, never a detail,
because they'll never see it there.

## Creating errors: the ladder

1. `errors.New("static message")` — the default.
2. `errors.Newf("invalid port: %d", port)` — when the message needs values.
3. `errors.Wrap(err, "context")` — adding context as an error travels up.
4. `errors.WithStack(err)` — re-returning a sentinel and you only want the stack.
5. `WithUserHint` / `WrapWithHint` — adding advice.

!!! danger "Never `fmt.Errorf`"
    `fmt.Errorf` captures **no stack trace**, so a failure that reaches your logs has
    no origin. Use `errors.Newf`/`errors.Wrap` from `cockroachdb/errors` instead —
    same ergonomics, plus a stack. (The handler tolerates plain errors; you just get
    no trace for them.)

## Prefer sentinels, wrap for specifics

Declare the conditions callers need to distinguish as package-level values, then wrap
for the dynamic part. That keeps `errors.Is` working while the message stays specific:

```go
var ErrInvalidPort = errors.New("invalid port: must be between 1 and 65535")

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
	return errorhandling.WithUserHint(err, "Run 'mytool init --github' to re-authenticate.")
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

Assertion failures are reported as `Internal error (assertion failure)` — the message
convention is to name the function, state the violated precondition, and say it's a
bug. No hint is appropriate: nobody outside your team can act on it.

## Related

- [The reporting model](../explanation/reporting-model.md)
- [Why cockroachdb/errors](../explanation/why-cockroachdb-errors.md)
- [Control the exit code](exit-codes.md)
