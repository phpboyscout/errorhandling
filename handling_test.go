package errorhandling

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/phpboyscout/go/errors"
)

func newTestHandler(t *testing.T) (*StandardErrorHandler, *CaptureLogger) {
	t.Helper()

	cap := NewCaptureLogger()

	return &StandardErrorHandler{Logger: cap.Logger()}, cap
}

// --- D8: Fatal reports and returns; it does not exit ------------------------

func TestFatalReturnsTheExitCodeAndDoesNotExit(t *testing.T) {
	t.Parallel()

	h, cap := newTestHandler(t)

	code := h.Fatal(context.Background(), WithExitCode(errors.New("boom"), 7))

	assert.Equal(t, 7, code, "Fatal must return the code, not act on it")
	require.Len(t, cap.Entries(), 1)
	assert.Equal(t, slog.LevelError, cap.Entries()[0].Level)
}

func TestFatalOfNilReportsNothing(t *testing.T) {
	t.Parallel()

	h, cap := newTestHandler(t)

	assert.Equal(t, 0, h.Fatal(context.Background(), nil))
	assert.Empty(t, cap.Entries())
}

func TestErrorAndWarnReportAtTheirLevel(t *testing.T) {
	t.Parallel()

	h, cap := newTestHandler(t)

	h.Error(context.Background(), errors.New("an error"))
	h.Warn(context.Background(), errors.New("a warning"))

	entries := cap.Entries()
	require.Len(t, entries, 2)
	assert.Equal(t, slog.LevelError, entries[0].Level)
	assert.Equal(t, slog.LevelWarn, entries[1].Level)
}

// --- D3: Quietly is a demotion, not a level ---------------------------------

func TestQuietlyDemotesTheLineButNotTheCode(t *testing.T) {
	t.Parallel()

	h, cap := newTestHandler(t)
	err := WithExitCode(errors.New("interrupted"), 130)

	code := h.Fatal(context.Background(), err, Quietly())

	assert.Equal(t, 130, code, "quiet must not change the exit code")
	require.Len(t, cap.Entries(), 1)
	assert.Equal(t, slog.LevelDebug, cap.Entries()[0].Level)
}

func TestWithPrefixLabelsTheReport(t *testing.T) {
	t.Parallel()

	h, cap := newTestHandler(t)

	h.Error(context.Background(), errors.New("boom"), WithPrefix("startup"))

	require.Len(t, cap.Entries(), 1)
	assert.Contains(t, cap.Entries()[0].Keyvals, KeyPrefix)
	assert.Contains(t, cap.Entries()[0].Keyvals, "startup")
}

// --- D4: the error is handed to slog, not taken apart -----------------------

func TestTheErrorArrivesAsAStructuredGroup(t *testing.T) {
	t.Parallel()

	h, cap := newTestHandler(t)

	err := errors.WithAttrs(
		errors.WithHint(errors.New("could not resolve"), "check the token"),
		slog.String("host", "codeberg.org"),
	)

	h.Error(context.Background(), err)

	group, ok := cap.ErrGroup()
	require.True(t, ok, "the error must arrive as a group, not a flattened string")
	assert.Equal(t, "could not resolve", group["msg"])
	assert.Equal(t, "codeberg.org", group["host"], "attributes travel on the error")
	assert.NotEmpty(t, group["hint"], "hints travel on the error")
	assert.NotEmpty(t, group["kind"], "the kind identifies the error")
}

func TestTheStackIsNotInTheGroupAndIsReachable(t *testing.T) {
	t.Parallel()

	h, cap := newTestHandler(t)
	err := errors.New("boom")

	h.Error(context.Background(), err)

	group, ok := cap.ErrGroup()
	require.True(t, ok)
	assert.NotContains(t, group, "stack", "LogValue omits the stack deliberately")
	assert.NotNil(t, errors.StackOf(err), "and it stays reachable through StackOf")
}

func TestTheStackIsAddedAtDebug(t *testing.T) {
	t.Parallel()

	h, cap := newTestHandler(t)
	cap.SetLevel(slog.LevelDebug)

	h.Error(context.Background(), errors.New("boom"))

	require.Len(t, cap.Entries(), 1)
	assert.Contains(t, cap.Entries()[0].Keyvals, KeyStacktrace)
}

func TestHelpIsAddedWhenConfigured(t *testing.T) {
	t.Parallel()

	cap := NewCaptureLogger()
	h := &StandardErrorHandler{Logger: cap.Logger(), Help: staticHelp("ask #support")}

	h.Error(context.Background(), errors.New("boom"))

	require.Len(t, cap.Entries(), 1)
	assert.Contains(t, cap.Entries()[0].Keyvals, KeyHelp)
	assert.Contains(t, cap.Entries()[0].Keyvals, "ask #support")
}

type staticHelp string

func (s staticHelp) SupportMessage() string { return string(s) }

// --- D5: an assertion failure is one record, identified by its kind ---------

func TestAssertionFailureIsReportedOnceWithItsKind(t *testing.T) {
	t.Parallel()

	h, cap := newTestHandler(t)

	err := NewAssertionFailure("invariant broken: %s", "x < 0")

	h.Error(context.Background(), err)

	require.Len(t, cap.Entries(), 1, "one record — the second line was the old way of saying this")
	assert.True(t, errors.Is(err, ErrAssertionFailure))

	group, ok := cap.ErrGroup()
	require.True(t, ok)
	assert.Equal(t, "errorhandling.assertion_failure", group["kind"],
		"the kind is what a query filters on")
}

// --- D1: the sentinels ------------------------------------------------------

func TestNewErrNotImplementedCarriesTheIssueURL(t *testing.T) {
	t.Parallel()

	err := NewErrNotImplemented("https://example.invalid/issues/1")

	require.True(t, errors.Is(err, ErrNotImplemented))

	attrs := errors.Attrs(err)
	require.Len(t, attrs, 1)
	assert.Equal(t, "issue_url", attrs[0].Key)
	assert.Equal(t, "https://example.invalid/issues/1", attrs[0].Value.String())
}

func TestNewErrNotImplementedWithoutAURLAddsNoAttribute(t *testing.T) {
	t.Parallel()

	err := NewErrNotImplemented("")

	require.True(t, errors.Is(err, ErrNotImplemented))
	assert.Empty(t, errors.Attrs(err))
}

// --- D2: the context ---------------------------------------------------------

func TestACancelledContextDoesNotChangeTheReport(t *testing.T) {
	t.Parallel()

	h, cap := newTestHandler(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	code := h.Fatal(ctx, WithExitCode(errors.New("boom"), 3))

	assert.Equal(t, 3, code)
	require.Len(t, cap.Entries(), 1, "a cancelled context must not suppress the report")
}
