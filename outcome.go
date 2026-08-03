package errorhandling

import (
	"log/slog"

	"gitlab.com/phpboyscout/go/errors"
)

// OutcomeKind identifies an attached [Outcome] to anything reading the error
// through the introspection contract — a log record, a span, a wire codec.
const OutcomeKind = "errorhandling.outcome"

// Outcome is how a terminal error should be presented and what exit code it
// yields.
//
// # Why this is data rather than a branch
//
// This module used to hold a closed switch over three sentinels it happened to
// know about. A consumer with its own "stop here, and here is what it means"
// error had nowhere to put it — which is why go-tool-base handled its
// update-complete case in its own execute.go instead, and why that case, the
// only one that exits ZERO, could not be expressed here at all.
//
// An outcome is declared beside the sentinel it describes, so the error carries
// its own disposition and any consumer can define one without touching this
// module.
type Outcome struct {
	// Code is the process exit code.
	//
	// Zero is legitimate and meaningful: an outcome can be terminal AND
	// successful. A completed self-update is exactly that — the run must stop,
	// and nothing went wrong.
	Code int

	// Level is how loudly to report it. A user-initiated stop is not an error
	// and an interrupt is not a failure, so neither should log like one.
	Level slog.Level

	// Message replaces the error's own text, for when that text is machinery
	// rather than something a user should read. Empty keeps the error's own.
	Message string

	// Usage prints the command's usage through the [ErrorHandler.SetUsage] seam
	// before reporting.
	//
	// This is the one presentation this module already owns, which is why it is
	// a bool rather than a callback: an Outcome carrying arbitrary behaviour
	// would make an error a place to hide control flow.
	Usage bool
}

// outcomeError attaches an Outcome without altering its cause's message or
// identity. Transparent to errors.Is and errors.As via Unwrap.
type outcomeError struct {
	cause   error
	outcome Outcome
}

func (e *outcomeError) Error() string     { return e.cause.Error() }
func (e *outcomeError) Unwrap() error     { return e.cause }
func (e *outcomeError) ErrorKind() string { return OutcomeKind }
func (e *outcomeError) ErrorPayload() any { return e.outcome }

// WithOutcome attaches a terminal disposition to err. Returns nil when err is
// nil, so it composes at a sentinel declaration without a guard.
//
//	var ErrUpdateComplete = errorhandling.WithOutcome(
//	    errors.NewSentinel("gtb.update_complete", "update complete — restart required"),
//	    errorhandling.Outcome{
//	        Code:    0,
//	        Level:   slog.LevelWarn,
//	        Message: "update complete — please run the command again",
//	    },
//	)
func WithOutcome(err error, o Outcome) error {
	if err == nil {
		return nil
	}

	return &outcomeError{cause: err, outcome: o}
}

// OutcomeOf returns the outermost Outcome in err's tree.
//
// Outermost wins: a caller wrapping someone else's error to change how it ends
// is making the more recent, more specific statement.
func OutcomeOf(err error) (Outcome, bool) {
	var found *outcomeError
	if errors.As(err, &found) {
		return found.outcome, true
	}

	return Outcome{}, false
}
