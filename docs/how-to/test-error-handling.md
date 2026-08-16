# Test error handling

Error paths are the ones least exercised by hand, so they earn their tests. This guide
covers the three things people trip over: asserting on the exit code, asserting on
structure rather than formatted strings, and mocking the handler.

## Fatal is safe to call in a test

Nothing in this module exits. `Fatal` reports and **returns** the code it believes the
process should use, so a test asserts on a plain `int` and needs no injected seam:

```go
h := errorhandling.New(slog.New(slog.DiscardHandler), nil)

code := h.Fatal(t.Context(), errorhandling.WithExitCode(errors.New("boom"), 3))
assert.Equal(t, 3, code)
```

The code an error *carries* and the code `Fatal` *returns* can differ — an
[`Outcome`](../reference/api.md#outcome) overrides an attached code — so assert on the
return value when the question is "what would the process do".

## Assert on hints, not on rendered text

Hints are structured data. Read them back rather than matching formatted output, which
is brittle:

```go
err := errors.WithHint(errors.New("token missing"),
	"Set the GITHUB_TOKEN environment variable")

assert.Contains(t, errors.Hints(err), "Set the GITHUB_TOKEN environment variable")
```

The same applies to exit codes — `errorhandling.ExitCode(err)` — and to identity:
`errors.Is(err, ErrThing)` keeps working through wraps, hints, and exit codes.

## Capture the output when you must

When the assertion really is about what got logged, point a slog handler at a buffer:

```go
var buf bytes.Buffer
h := errorhandling.New(slog.New(slog.NewTextHandler(&buf, nil)), nil)

h.Error(t.Context(), errors.New("something went wrong"))
assert.Contains(t, buf.String(), "something went wrong")
```

To exercise the debug-gated output — the stack trace — enable debug on the handler you
pass in:

```go
logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
```

The handler decides by asking the logger (`Enabled(ctx, slog.LevelDebug)`), so whatever
your handler considers debug is what governs.

## Mock the handler

To assert that code under test *reported* something — without caring how it rendered —
use the published mocks:

```go
import ehmocks "gitlab.com/phpboyscout/go/errorhandling/mocks"

func TestReportsFailure(t *testing.T) {
	h := ehmocks.NewMockErrorHandler(t)
	h.EXPECT().Error(mock.Anything, mock.Anything).Once()

	runThing(h) // takes an errorhandling.ErrorHandler

	// expectations are verified on cleanup
}
```

`MockHelpConfig` is available for the same reason. Aliasing the import (`ehmocks` here)
is the convention — the package is named `mocks`, so aliasing keeps it unambiguous
alongside other modules' mocks.

## Testing an error's own behaviour

For errors your code produces, assert the properties, not the prose:

```go
err := loadConfig("/nonexistent.yaml")

require.Error(t, err)
assert.ErrorIs(t, err, ErrConfigMissing)                  // identity survives wrapping
assert.Contains(t, err.Error(), "failed to read config")  // context was added
assert.NotNil(t, errors.StackOf(err))                     // a stack was captured
```

That last one is worth having somewhere: it's what catches a `fmt.Errorf` sneaking in,
since a plain error carries no stack.

## Related

- [Control the exit code](exit-codes.md)
- [Write actionable errors](actionable-errors.md)
