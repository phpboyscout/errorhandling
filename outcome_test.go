package errorhandling_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/phpboyscout/go/errors"

	"gitlab.com/phpboyscout/go/errorhandling"
)

// TestOutcomeCanBeTerminalAndSuccessful is the case the old closed switch could
// not express at all, and the reason go-tool-base handled its update-complete
// error in its own execute.go instead.
func TestOutcomeCanBeTerminalAndSuccessful(t *testing.T) {
	t.Parallel()

	buf := errorhandling.NewCaptureLogger()
	h := errorhandling.New(buf.Logger(), nil)

	err := errorhandling.WithOutcome(
		errors.NewSentinel("test.update_complete", "update complete — restart required"),
		errorhandling.Outcome{
			Code:    0,
			Level:   slog.LevelWarn,
			Message: "update complete — please run the command again",
		},
	)

	code := h.Fatal(context.Background(), err)

	assert.Equal(t, 0, code, "an outcome may be terminal and still succeed")

	entries := buf.Entries()
	require.Len(t, entries, 1)
	assert.Equal(t, slog.LevelWarn, entries[0].Level)
	assert.Equal(t, "update complete — please run the command again", entries[0].Message,
		"the outcome's message replaces the error's own")
}

func TestOutcomeIsTransparentToIs(t *testing.T) {
	t.Parallel()

	sentinel := errors.NewSentinel("test.stop", "stop")
	err := errorhandling.WithOutcome(sentinel, errorhandling.Outcome{Code: 3})

	assert.True(t, errors.Is(err, sentinel))
	assert.True(t, errors.Is(errors.Wrap(err, "context"), sentinel),
		"and through further wrapping")
}

func TestOutcomeBeatsWithExitCode(t *testing.T) {
	t.Parallel()

	h := errorhandling.New(slog.New(slog.DiscardHandler), nil)

	err := errorhandling.WithOutcome(
		errorhandling.WithExitCode(errors.New("boom"), 7),
		errorhandling.Outcome{Code: 2, Level: slog.LevelWarn},
	)

	assert.Equal(t, 2, h.Fatal(context.Background(), err),
		"the outcome is the more specific statement")
}

func TestOutcomeOverridesTheCallersLevel(t *testing.T) {
	t.Parallel()

	buf := errorhandling.NewCaptureLogger()
	h := errorhandling.New(buf.Logger(), nil)

	err := errorhandling.WithOutcome(errors.New("stop"),
		errorhandling.Outcome{Code: 0, Level: slog.LevelWarn})

	// Reported via Error, but the error knows it is only a warning.
	h.Error(context.Background(), err)

	require.Len(t, buf.Entries(), 1)
	assert.Equal(t, slog.LevelWarn, buf.Entries()[0].Level)
}

func TestOutcomeUsagePrintsThroughTheSeam(t *testing.T) {
	t.Parallel()

	h := errorhandling.New(slog.New(slog.DiscardHandler), nil)

	printed := false
	h.SetUsage(func() error { printed = true; return nil })

	code := h.Fatal(context.Background(), errorhandling.ErrRunSubCommand)

	assert.True(t, printed, "an outcome asking for usage prints through SetUsage")
	assert.Equal(t, errorhandling.ExitCodeUsage, code)
}

func TestSentinelsCarryTheirOutcomes(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		err   error
		code  int
		level slog.Level
		usage bool
	}{
		"not implemented": {errorhandling.ErrNotImplemented, errorhandling.ExitCodeUsage, slog.LevelWarn, false},
		"run subcommand":  {errorhandling.ErrRunSubCommand, errorhandling.ExitCodeUsage, slog.LevelWarn, true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			outcome, ok := errorhandling.OutcomeOf(tc.err)
			require.True(t, ok, "this module's terminal sentinels declare their own outcome")
			assert.Equal(t, tc.code, outcome.Code)
			assert.Equal(t, tc.level, outcome.Level)
			assert.Equal(t, tc.usage, outcome.Usage)
		})
	}
}

func TestNoOutcomeLeavesTheCallerInCharge(t *testing.T) {
	t.Parallel()

	_, ok := errorhandling.OutcomeOf(errors.New("plain"))
	assert.False(t, ok)

	h := errorhandling.New(slog.New(slog.DiscardHandler), nil)
	assert.Equal(t, 1, h.Fatal(context.Background(), errors.New("plain")),
		"an error with no outcome and no code falls back to 1")
}
