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
		"not implemented":    {errorhandling.ErrNotImplemented, errorhandling.ExitCodeUsage, slog.LevelWarn, false},
		"run subcommand":     {errorhandling.ErrRunSubCommand, errorhandling.ExitCodeUsage, slog.LevelWarn, true},
		"unknown subcommand": {errorhandling.ErrUnknownSubCommand, errorhandling.ExitCodeUsage, slog.LevelWarn, true},
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

// TestWrappingASentinelKeepsItsIdentityAndOutcome pins the shape every caller of
// ErrUnknownSubCommand uses: the sentinel says what kind of failure it is and
// how the process should end, and the wrap says which verb caused it. Both have
// to survive together, or the caller has to choose between a useful message and
// a correct exit code.
func TestWrappingASentinelKeepsItsIdentityAndOutcome(t *testing.T) {
	t.Parallel()

	err := errors.Wrapf(errorhandling.ErrUnknownSubCommand,
		"unknown command %q for %q", "bogus", "tool alpha")

	require.ErrorIs(t, err, errorhandling.ErrUnknownSubCommand)

	outcome, ok := errorhandling.OutcomeOf(err)
	require.True(t, ok, "an outcome is reachable through a wrap, not only on the bare sentinel")
	assert.Equal(t, errorhandling.ExitCodeUsage, outcome.Code)
	assert.True(t, outcome.Usage)

	// The wrap's own text has to survive to the reported line. A caller that
	// names the offending verb and then sees only "unknown subcommand" logged
	// has gained nothing over the bare sentinel.
	buf := errorhandling.NewCaptureLogger()
	h := errorhandling.New(buf.Logger(), nil)

	printed := false
	h.SetUsage(func() error { printed = true; return nil })

	code := h.Fatal(context.Background(), err)

	require.Len(t, buf.Entries(), 1)
	assert.Equal(t, `unknown command "bogus" for "tool alpha": unknown subcommand`,
		buf.Entries()[0].Message)
	assert.Equal(t, slog.LevelWarn, buf.Entries()[0].Level)
	assert.True(t, printed, "the outcome asks for usage")
	assert.Equal(t, errorhandling.ExitCodeUsage, code)
}

func TestNoOutcomeLeavesTheCallerInCharge(t *testing.T) {
	t.Parallel()

	_, ok := errorhandling.OutcomeOf(errors.New("plain"))
	assert.False(t, ok)

	h := errorhandling.New(slog.New(slog.DiscardHandler), nil)
	assert.Equal(t, 1, h.Fatal(context.Background(), errors.New("plain")),
		"an error with no outcome and no code falls back to 1")
}

// An outcome says how an error ENDS, not what it is. Attaching one used to
// replace the error's identity in every log record and query — KindOf reported
// errorhandling.outcome for all three sentinels this module ships, which is the
// plumbing, on exactly the errors most likely to be queried.
//
// go/errors gained StructuralKinder for this: a wrapper can say its kind is an
// annotation and KindOf looks past it.
func TestAttachmentsDoNotMaskTheErrorsIdentity(t *testing.T) {
	t.Parallel()

	for name, err := range map[string]error{
		"outcome":   errorhandling.ErrRunSubCommand,
		"unknown":   errorhandling.ErrUnknownSubCommand,
		"not impl":  errorhandling.ErrNotImplemented,
		"exit code": errorhandling.WithExitCode(errors.NewSentinel("app.boom", "boom"), 3),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			kind := errors.KindOf(err)

			assert.NotEqual(t, errorhandling.OutcomeKind, kind,
				"an outcome is how the error ends, not what it is")
			assert.NotEqual(t, errorhandling.ExitCodeKind, kind,
				"and neither is an exit code")
			assert.NotEmpty(t, kind, "the identity beneath is still reported")
		})
	}
}

func TestTheSentinelsReportTheirOwnKinds(t *testing.T) {
	t.Parallel()

	for kind, err := range map[string]error{
		"errorhandling.run_subcommand":     errorhandling.ErrRunSubCommand,
		"errorhandling.unknown_subcommand": errorhandling.ErrUnknownSubCommand,
		"errorhandling.not_implemented":    errorhandling.ErrNotImplemented,
		"errorhandling.assertion_failure":  errorhandling.ErrAssertionFailure,
	} {
		assert.Equal(t, kind, errors.KindOf(err),
			"this is what a log query filters on")
	}
}

// UnknownSubCommand is the shared half of reporting a mistyped verb. The cobra
// closure that calls it cannot be shared — this module bans cobra, and a module
// for eight lines of glue would not earn its keep — but the message and the
// sentinel must not drift between go-tool-base's generated groups and the
// standalone CLIs that build their own.
func TestUnknownSubCommandCarriesTheVerbAndTheSentinel(t *testing.T) {
	t.Parallel()

	err := errorhandling.UnknownSubCommand("bogus", "tool alpha")

	require.ErrorIs(t, err, errorhandling.ErrUnknownSubCommand)
	assert.Equal(t, `unknown command "bogus" for "tool alpha": unknown subcommand`, err.Error())

	outcome, ok := errorhandling.OutcomeOf(err)
	require.True(t, ok)
	assert.Equal(t, errorhandling.ExitCodeUsage, outcome.Code)
	assert.True(t, outcome.Usage)
}
