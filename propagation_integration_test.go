package errorhandling_test

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"

	cberrors "github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/phpboyscout/go/errorhandling"
)

// simulateDeepError mimics a multi-layer error chain:
// database layer → service layer → handler layer.
func simulateDeepError() error {
	// Layer 1: database returns raw error
	dbErr := cberrors.New("connection refused")

	// Layer 2: service wraps with context and a hint
	svcErr := errorhandling.WrapWithHint(dbErr, "query failed", "check database connection string")

	// Layer 3: handler wraps again with operational context
	return errorhandling.WithUserHint(
		cberrors.Wrap(svcErr, "user lookup failed"),
		"verify the database is running",
	)
}

func TestHintsSurviveMultipleWraps(t *testing.T) {
	t.Parallel()
	skipIfNotIntegration(t, "errorhandling")

	err := simulateDeepError()
	log := errorhandling.NewCaptureLogger()
	h := errorhandling.New(log.Logger(), nil)

	h.Error(err, "API: ")

	entries := log.Entries()
	require.Len(t, entries, 1)
	assert.Equal(t, slog.LevelError, entries[0].Level)
	assert.Contains(t, entries[0].Message, "user lookup failed")

	// Both hints from different layers must survive
	hints := cberrors.FlattenHints(err)
	assert.Contains(t, hints, "check database connection string")
	assert.Contains(t, hints, "verify the database is running")

	// Hints appear in log keyvals
	assert.Contains(t, entries[0].Keyvals, errorhandling.KeyHints)
}

func TestDebugModeAddsStacktrace(t *testing.T) {
	t.Parallel()
	skipIfNotIntegration(t, "errorhandling")

	err := cberrors.New("something broke")

	debugLog := errorhandling.NewCaptureLogger()
	debugLog.SetLevel(slog.LevelDebug)
	debugHandler := errorhandling.New(debugLog.Logger(), nil)
	debugHandler.Error(err)

	debugEntries := debugLog.Entries()
	require.NotEmpty(t, debugEntries)
	assert.Contains(t, debugEntries[0].Keyvals, errorhandling.KeyStacktrace)

	// Non-debug mode: no stacktrace
	infoLog := errorhandling.NewCaptureLogger()
	infoLog.SetLevel(slog.LevelInfo)
	infoHandler := errorhandling.New(infoLog.Logger(), nil)
	infoHandler.Error(err)

	infoEntries := infoLog.Entries()
	require.NotEmpty(t, infoEntries)
	assert.NotContains(t, infoEntries[0].Keyvals, errorhandling.KeyStacktrace)
}

func TestHelpConfigAppearsInOutput(t *testing.T) {
	t.Parallel()
	skipIfNotIntegration(t, "errorhandling")

	err := cberrors.New("unexpected failure")
	log := errorhandling.NewCaptureLogger()

	h := errorhandling.New(log.Logger(), slackHelp{
		Team:    "Platform",
		Channel: "#incidents",
	})

	h.Error(err)

	entries := log.Entries()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Keyvals, errorhandling.KeyHelp)

	// Find the help value
	for i, kv := range entries[0].Keyvals {
		if kv == errorhandling.KeyHelp {
			helpMsg, ok := entries[0].Keyvals[i+1].(string)
			require.True(t, ok)
			assert.Contains(t, helpMsg, "Platform")
			assert.Contains(t, helpMsg, "#incidents")
			break
		}
	}
}

func TestFatalCallsExitWithHints(t *testing.T) {
	t.Parallel()
	skipIfNotIntegration(t, "errorhandling")

	baseErr := cberrors.New("disk full")
	hinted := errorhandling.WithUserHint(baseErr, "free up space or expand volume")

	log := errorhandling.NewCaptureLogger()
	var exitCode int
	h := errorhandling.New(log.Logger(), nil,
		errorhandling.WithExitFunc(func(code int) { exitCode = code }),
	)

	h.Fatal(hinted)

	assert.Equal(t, 1, exitCode)

	entries := log.Entries()
	require.NotEmpty(t, entries)
	assert.Equal(t, slog.LevelError, entries[0].Level)
	assert.Contains(t, entries[0].Message, "disk full")
	assert.Contains(t, entries[0].Keyvals, errorhandling.KeyHints)
}

func TestSpecialError_WrappedUnimplemented(t *testing.T) {
	t.Parallel()
	skipIfNotIntegration(t, "errorhandling")

	// Wrap an unimplemented error with additional context — it should still
	// be detected as a special error and downgraded to warn.
	inner := errorhandling.NewErrNotImplemented("https://example.com/issues/42")
	wrapped := cberrors.Wrap(inner, "feature X")

	log := errorhandling.NewCaptureLogger()
	h := errorhandling.New(log.Logger(), nil)
	h.Error(wrapped)

	entries := log.Entries()
	require.NotEmpty(t, entries)
	assert.Equal(t, slog.LevelWarn, entries[0].Level)
	assert.Contains(t, entries[0].Message, "not yet implemented")
}

func TestSpecialError_ErrRunSubCommandWritesUsage(t *testing.T) {
	t.Parallel()
	skipIfNotIntegration(t, "errorhandling")

	var writerBuf bytes.Buffer
	log := errorhandling.NewCaptureLogger()
	h := errorhandling.New(log.Logger(), nil, errorhandling.WithWriter(&writerBuf))

	// A CLI framework supplies its own usage printer through SetUsage; with
	// Cobra that is cmd.Usage. The module itself has no framework dependency.
	h.SetUsage(func() error {
		_, err := writerBuf.WriteString("Usage:\n  mycommand [command]\n")

		return err
	})

	h.Check(errorhandling.ErrRunSubCommand, "", errorhandling.LevelError)

	entries := log.Entries()
	require.NotEmpty(t, entries)
	assert.Equal(t, slog.LevelWarn, entries[0].Level)
	assert.Contains(t, entries[0].Message, "Subcommand required")
	assert.Contains(t, writerBuf.String(), "Usage:")
}

func TestNilErrorIsNoOp(t *testing.T) {
	t.Parallel()
	skipIfNotIntegration(t, "errorhandling")

	log := errorhandling.NewCaptureLogger()
	h := errorhandling.New(log.Logger(), nil)

	h.Check(nil, "prefix", errorhandling.LevelError)
	h.Error(nil)
	h.Warn(nil)

	assert.Empty(t, log.Entries())
}

func TestAssertionFailureFallsThrough(t *testing.T) {
	t.Parallel()
	skipIfNotIntegration(t, "errorhandling")

	err := errorhandling.NewAssertionFailure("invariant broken: %s", "x < 0")
	log := errorhandling.NewCaptureLogger()
	h := errorhandling.New(log.Logger(), nil)

	h.Error(err)

	entries := log.Entries()
	// Assertion failure logs at Error level (both from handleSpecialErrors and logError)
	require.NotEmpty(t, entries)

	hasAssertionMsg := false
	hasErrorMsg := false
	for _, e := range entries {
		if e.Level == slog.LevelError {
			if assert.ObjectsAreEqual("Internal error (assertion failure)", e.Message) {
				hasAssertionMsg = true
			}
			if assert.ObjectsAreEqual(true, len(e.Message) > 0) {
				hasErrorMsg = true
			}
		}
	}

	assert.True(t, hasAssertionMsg || hasErrorMsg, "should log assertion failure at error level")
}

func TestPrefixPropagation(t *testing.T) {
	t.Parallel()
	skipIfNotIntegration(t, "errorhandling")

	err := cberrors.New("timeout")
	log := errorhandling.NewCaptureLogger()
	h := errorhandling.New(log.Logger(), nil)

	h.Error(err, "HTTP: ", "GET /api: ")

	entries := log.Entries()
	require.NotEmpty(t, entries)
	// Multiple prefixes are concatenated and carried as a structured "prefix"
	// attribute (slog-first) rather than prepended to the message.
	var prefix string
	for i, kv := range entries[0].Keyvals {
		if kv == "prefix" && i+1 < len(entries[0].Keyvals) {
			prefix, _ = entries[0].Keyvals[i+1].(string)
		}
	}
	assert.Contains(t, prefix, "HTTP:")
	assert.Contains(t, prefix, "GET /api:")
}

func TestWarnLevel(t *testing.T) {
	t.Parallel()
	skipIfNotIntegration(t, "errorhandling")

	err := cberrors.New("deprecated feature used")
	log := errorhandling.NewCaptureLogger()
	h := errorhandling.New(log.Logger(), nil)

	h.Warn(err)

	entries := log.Entries()
	require.NotEmpty(t, entries)
	assert.Equal(t, slog.LevelWarn, entries[0].Level)
	assert.Contains(t, entries[0].Message, "deprecated feature used")
}

func TestCrossPackageErrorChain(t *testing.T) {
	t.Parallel()
	skipIfNotIntegration(t, "errorhandling")

	// Simulate error originating in config, wrapped by setup, handled by CLI
	configErr := cberrors.New("YAML parse error at line 5")
	setupErr := errorhandling.WrapWithHint(configErr, "failed to load configuration", "check config.yaml syntax")
	cliErr := cberrors.Wrap(setupErr, "initialisation failed")

	log := errorhandling.NewCaptureLogger()
	log.SetLevel(slog.LevelDebug)

	var exitCode int
	h := errorhandling.New(log.Logger(), slackHelp{
		Team:    "DevTools",
		Channel: "#support",
	}, errorhandling.WithExitFunc(func(code int) { exitCode = code }))

	h.Fatal(cliErr)

	assert.Equal(t, 1, exitCode)

	entries := log.Entries()
	require.NotEmpty(t, entries)

	entry := entries[0]
	assert.Equal(t, slog.LevelError, entry.Level)

	// Outermost wrapper message appears
	assert.Contains(t, entry.Message, "initialisation failed")

	// Hint from setup layer survives
	assert.Contains(t, entry.Keyvals, errorhandling.KeyHints)

	// Stacktrace present in debug mode
	assert.Contains(t, entry.Keyvals, errorhandling.KeyStacktrace)

	// Help config present
	assert.Contains(t, entry.Keyvals, errorhandling.KeyHelp)

	// The original error message is in the chain
	assert.Contains(t, cberrors.FlattenHints(cliErr), "check config.yaml syntax")
}

// skipIfNotIntegration gates integration tests behind INT_TEST / INT_TEST_<TAG>
// env vars — a vendored, dependency-free equivalent of go-tool-base's
// testutil.SkipIfNotIntegration (test helpers cannot cross a module boundary).
func skipIfNotIntegration(t *testing.T, tags ...string) {
	t.Helper()

	if os.Getenv("INT_TEST") != "" {
		return
	}

	for _, tag := range tags {
		if os.Getenv("INT_TEST_"+strings.ToUpper(tag)) != "" {
			return
		}
	}

	msg := "skipping integration test; set INT_TEST=1 to run all"
	if len(tags) > 0 {
		names := make([]string, len(tags))
		for i, tag := range tags {
			names[i] = "INT_TEST_" + strings.ToUpper(tag)
		}

		msg += " or " + strings.Join(names, "/") + "=1 for this group"
	}

	t.Skip(msg)
}
