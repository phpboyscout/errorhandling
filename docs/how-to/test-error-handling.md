# Test error handling

Error paths are the ones least exercised by hand, so they earn their tests. This guide
covers the three things people trip over: not killing the test binary, asserting on
structure rather than formatted strings, and mocking the handler.

## Never let Fatal kill your tests

`Fatal` calls the exit function, which is `os.Exit` by default — in a test, that takes
the whole test binary with it. Inject a fake:

```go
var code int
h := errorhandling.New(slog.New(slog.DiscardHandler), nil,
	errorhandling.WithExitFunc(func(c int) { code = c }))

h.Fatal(errorhandling.WithExitCode(errors.New("boom"), 3))
assert.Equal(t, 3, code)
```

!!! warning "With a fake exit func, execution continues past Fatal"
    Real `os.Exit` never returns; your fake does. So a test that calls `Fatal` keeps
    running the following lines — including any code that assumed it had terminated.
    Return or guard explicitly if that matters.

## Assert on hints, not on rendered text

Hints are structured data. Read them back rather than matching formatted output, which
is brittle:

```go
err := errorhandling.WithUserHint(errors.New("token missing"),
	"Set the GITHUB_TOKEN environment variable")

assert.Contains(t, errors.FlattenHints(err), "GITHUB_TOKEN")
```

The same applies to exit codes — `errorhandling.ExitCode(err)` — and to identity:
`errors.Is(err, ErrThing)` keeps working through wraps, hints, and exit codes.

## Capture the output when you must

When the assertion really is about what got logged, point a slog handler at a buffer:

```go
var buf bytes.Buffer
h := errorhandling.New(slog.New(slog.NewTextHandler(&buf, nil)), nil)

h.Error(errors.New("something went wrong"))
assert.Contains(t, buf.String(), "something went wrong")
```

To exercise the debug-gated output — stack traces and details — enable debug on the
handler you pass in:

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
	h.EXPECT().Error(mock.Anything).Once()

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
assert.NotEmpty(t, fmt.Sprintf("%+v", err))               // a stack was captured
```

That last one is worth having somewhere: it's what catches a `fmt.Errorf` sneaking in,
since a plain error carries no stack.

## Related

- [Control the exit code](exit-codes.md)
- [Write actionable errors](actionable-errors.md)
