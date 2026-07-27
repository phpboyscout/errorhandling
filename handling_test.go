package errorhandling

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"testing"

	cberrors "github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorHandler_Check(t *testing.T) {
	t.Run("Error_logs_message_with_prefix", func(t *testing.T) {
		log := NewCaptureLogger()
		h := &StandardErrorHandler{Logger: log.Logger(), Exit: os.Exit}
		h.Error(errors.New("simple error"), "Prefix: ")
		entries := log.Entries()
		require.NotEmpty(t, entries)
		assert.Contains(t, entries[0].Message, "simple error")
		// The prefix is now carried as a structured attribute rather than
		// prepended to the message (slog-first migration).
		assert.Contains(t, entries[0].Keyvals, "prefix")
		assert.Contains(t, entries[0].Keyvals, "Prefix: ")
	})

	t.Run("Warn_logs_warning", func(t *testing.T) {
		log := NewCaptureLogger()
		h := &StandardErrorHandler{Logger: log.Logger(), Exit: os.Exit}
		h.Warn(errors.New("simple warning"), "Prefix: ")
		entries := log.Entries()
		require.NotEmpty(t, entries)
		assert.Contains(t, entries[0].Message, "simple warning")
	})

	t.Run("Unknown_level_falls_back_to_error", func(t *testing.T) {
		log := NewCaptureLogger()
		h := &StandardErrorHandler{Logger: log.Logger(), Exit: os.Exit}
		h.Check(errors.New("boom"), "", "totally-unknown-level")
		entries := log.Entries()
		require.NotEmpty(t, entries, "unknown level must not silently swallow the error")
		assert.Equal(t, slog.LevelError, entries[0].Level)
		assert.Contains(t, entries[0].Message, "boom")
	})

	t.Run("ErrNotImplemented_downgrades_to_warn", func(t *testing.T) {
		log := NewCaptureLogger()
		h := &StandardErrorHandler{Logger: log.Logger(), Exit: os.Exit}
		h.Check(ErrNotImplemented, "", LevelError)
		entries := log.Entries()
		require.NotEmpty(t, entries)
		assert.Equal(t, slog.LevelWarn, entries[0].Level)
		assert.Contains(t, entries[0].Message, "Command not yet implemented")
	})

	t.Run("ErrRunSubCommand_with_usage_property", func(t *testing.T) {
		var writerBuf bytes.Buffer
		log := NewCaptureLogger()
		h := &StandardErrorHandler{Logger: log.Logger(), Exit: os.Exit}
		// The usage seam is a plain func — no CLI framework involved. A real
		// Cobra caller supplies cmd.Usage here.
		h.SetUsage(func() error {
			_, err := writerBuf.WriteString("Usage:\n  testcmd [command]\n")

			return err
		})
		h.Check(ErrRunSubCommand, "", LevelError)
		entries := log.Entries()
		require.NotEmpty(t, entries)
		assert.Equal(t, slog.LevelWarn, entries[0].Level)
		assert.Contains(t, entries[0].Message, "Subcommand required")
		assert.Contains(t, writerBuf.String(), "Usage:")
	})

	t.Run("ErrRunSubCommand_via_Error_wrapper", func(t *testing.T) {
		var writerBuf bytes.Buffer
		log := NewCaptureLogger()
		h := &StandardErrorHandler{Logger: log.Logger(), Exit: os.Exit}
		// The usage seam is a plain func — no CLI framework involved. A real
		// Cobra caller supplies cmd.Usage here.
		h.SetUsage(func() error {
			_, err := writerBuf.WriteString("Usage:\n  testcmd [command]\n")

			return err
		})
		h.Error(ErrRunSubCommand)
		entries := log.Entries()
		require.NotEmpty(t, entries)
		assert.Equal(t, slog.LevelWarn, entries[0].Level)
		assert.Contains(t, entries[0].Message, "Subcommand required")
		assert.Contains(t, writerBuf.String(), "Usage:")
	})
}

func TestNewErrNotImplemented(t *testing.T) {
	t.Parallel()
	err := NewErrNotImplemented("https://github.com/org/repo/issues/1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
	links := cberrors.GetAllIssueLinks(err)
	require.Len(t, links, 1)
	assert.Equal(t, "https://github.com/org/repo/issues/1", links[0].IssueURL)
}

func TestHandleSpecialErrors_UnimplementedWithIssueLink(t *testing.T) {
	t.Parallel()
	log := NewCaptureLogger()
	h := &StandardErrorHandler{Logger: log.Logger(), Exit: os.Exit}

	err := NewErrNotImplemented("https://example.com/issue/99")
	handled := h.handleSpecialErrors(err)
	assert.True(t, handled)
	entries := log.Entries()
	require.Len(t, entries, 2)
	assert.Contains(t, entries[0].Message, "not yet implemented")
	assert.Contains(t, entries[1].Keyvals, "https://example.com/issue/99")
}

func TestHandleSpecialErrors_AssertionFailure(t *testing.T) {
	t.Parallel()
	log := NewCaptureLogger()
	h := &StandardErrorHandler{Logger: log.Logger(), Exit: os.Exit}

	err := NewAssertionFailure("invariant violated: %s", "x must be positive")
	handled := h.handleSpecialErrors(err)
	assert.False(t, handled) // assertion failures fall through to logError
	entries := log.Entries()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "Internal error")
}

func TestHandleSpecialErrors_ErrRunSubCommand_NilCmd(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := &StandardErrorHandler{Logger: slog.New(slog.NewTextHandler(&buf, nil)), Exit: os.Exit}

	// No Usage set — still returns true (nothing to print, but handled)
	handled := h.handleSpecialErrors(ErrRunSubCommand)
	assert.True(t, handled)
}

func TestWithUserHint(t *testing.T) {
	t.Parallel()
	base := cberrors.New("base error")
	hinted := WithUserHint(base, "try restarting")
	assert.Contains(t, cberrors.FlattenHints(hinted), "try restarting")
}

func TestWithUserHintf(t *testing.T) {
	t.Parallel()
	base := cberrors.New("base error")
	hinted := WithUserHintf(base, "try %s", "again")
	assert.Contains(t, cberrors.FlattenHints(hinted), "try again")
}

func TestWrapWithHint(t *testing.T) {
	t.Parallel()
	base := cberrors.New("root cause")
	wrapped := WrapWithHint(base, "operation failed", "check your config")
	assert.Contains(t, wrapped.Error(), "operation failed")
	assert.Contains(t, cberrors.FlattenHints(wrapped), "check your config")
}

func TestNewAssertionFailure(t *testing.T) {
	t.Parallel()
	err := NewAssertionFailure("unexpected state: %d", 42)
	require.Error(t, err)
	assert.True(t, cberrors.HasAssertionFailure(err))
	assert.Contains(t, err.Error(), "unexpected state: 42")
}

func TestErrorHandler_Fatal(t *testing.T) {
	log := NewCaptureLogger()

	exitCalled := false
	exitCode := 0
	mockExit := func(code int) {
		exitCalled = true
		exitCode = code
	}

	h := &StandardErrorHandler{
		Logger: log.Logger(),
		Exit:   mockExit,
	}

	err := errors.New("fatal error")
	h.Fatal(err, "FATAL: ")

	assert.True(t, exitCalled)
	assert.Equal(t, 1, exitCode)
	entries := log.Entries()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "fatal error")
}

// TestErrorHandler_Fatal_SpecialErrors proves that a fatal-level special error
// (subcommand-required / not-implemented) both terminates the process with the
// usage exit code (2) and still emits its special-error presentation. It also
// asserts that reporting the same special errors at a non-fatal level does not
// exit. On unmodified main this fails: Check returns early inside
// handleSpecialErrors, so h.Exit is never reached for a fatal special error.
func TestErrorHandler_Fatal_SpecialErrors(t *testing.T) {
	tests := []struct {
		name         string
		newErr       func(buf *bytes.Buffer) error
		level        string
		wantExit     bool
		wantCode     int
		wantMsgFrag  string
		wantMsgLevel slog.Level
	}{
		{
			name:         "fatal_ErrRunSubCommand_exits_2_and_prints_usage",
			newErr:       func(*bytes.Buffer) error { return ErrRunSubCommand },
			level:        LevelFatal,
			wantExit:     true,
			wantCode:     ExitCodeUsage,
			wantMsgFrag:  "Subcommand required",
			wantMsgLevel: slog.LevelWarn,
		},
		{
			name:         "fatal_ErrNotImplemented_exits_2_and_reports",
			newErr:       func(*bytes.Buffer) error { return NewErrNotImplemented("https://example.com/issue/1") },
			level:        LevelFatal,
			wantExit:     true,
			wantCode:     ExitCodeUsage,
			wantMsgFrag:  "not yet implemented",
			wantMsgLevel: slog.LevelWarn,
		},
		{
			name:         "error_ErrRunSubCommand_does_not_exit",
			newErr:       func(*bytes.Buffer) error { return ErrRunSubCommand },
			level:        LevelError,
			wantExit:     false,
			wantMsgFrag:  "Subcommand required",
			wantMsgLevel: slog.LevelWarn,
		},
		{
			name:         "warn_ErrNotImplemented_does_not_exit",
			newErr:       func(*bytes.Buffer) error { return NewErrNotImplemented("https://example.com/issue/2") },
			level:        LevelWarn,
			wantExit:     false,
			wantMsgFrag:  "not yet implemented",
			wantMsgLevel: slog.LevelWarn,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			log := NewCaptureLogger()

			exitCalled := false
			exitCode := -1
			mockExit := func(code int) {
				exitCalled = true
				exitCode = code
			}

			var usageBuf bytes.Buffer
			h := &StandardErrorHandler{Logger: log.Logger(), Exit: mockExit}
			h.SetUsage(func() error {
				_, err := usageBuf.WriteString("Usage:\n  testcmd [command]\n")

				return err
			})

			h.Check(tc.newErr(&usageBuf), "", tc.level)

			assert.Equal(t, tc.wantExit, exitCalled, "exit-called expectation")
			if tc.wantExit {
				assert.Equal(t, tc.wantCode, exitCode, "exit code")
			}

			// Presentation must survive regardless of level.
			entries := log.Entries()
			require.NotEmpty(t, entries, "special-error presentation must still be emitted")
			assert.Equal(t, tc.wantMsgLevel, entries[0].Level)
			assert.Contains(t, entries[0].Message, tc.wantMsgFrag)
		})
	}
}
