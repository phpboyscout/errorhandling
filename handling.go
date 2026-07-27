package errorhandling

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/cockroachdb/errors"
)

const (
	LevelFatal = "fatal"
	// LevelFatalQuiet exits the process exactly like LevelFatal — honouring any
	// exit code attached via WithExitCode — but logs the message at debug rather
	// than error. It exists for expected, user-initiated terminations (e.g. a
	// SIGINT/SIGTERM interrupt) where the non-zero exit code is the signal and an
	// error-level log line would be noise. The message is still emitted at debug
	// so `--debug` continues to surface it.
	LevelFatalQuiet = "fatal-quiet"
	LevelError      = "error"
	LevelWarn       = "warn"
	KeyStacktrace   = "stacktrace"
	KeyHelp         = "help"
	KeyHints        = "hints"
	KeyDetails      = "details"
)

var (
	ErrNotImplemented = errors.New("command not yet implemented")
	ErrRunSubCommand  = errors.New("subcommand required")
)

// ExitFunc is the function called to terminate the process. Defaults to os.Exit.
// Override via WithExitFunc for testing.
type ExitFunc func(code int)

// ErrorHandler defines the interface for structured error reporting. It
// formats errors with hints, stack traces, and help channel information,
// then routes them to the appropriate output (logger, writer, or exit).
type ErrorHandler interface {
	Check(err error, prefix string, level string)
	Fatal(err error, prefixes ...string)
	Error(err error, prefixes ...string)
	Warn(err error, prefixes ...string)

	// SetUsage registers the function used to print usage when an error
	// wrapping [ErrRunSubCommand] is reported. CLI frameworks supply their
	// own printer — with Cobra, that is `SetUsage(cmd.Usage)`, typically set
	// in each command's pre-run so the usage shown belongs to the command
	// that actually failed.
	SetUsage(usage func() error)
}

// StandardErrorHandler is the default ErrorHandler implementation.
// It extracts hints, details, and stack traces from cockroachdb/errors
// and formats them for terminal or structured output. All error output is
// routed through Logger; usage output for ErrRunSubCommand goes through the
// Usage seam.
type StandardErrorHandler struct {
	Logger *slog.Logger
	Help   HelpConfig
	Exit   ExitFunc
	Usage  func() error
}

// New creates an ErrorHandler with the given logger and help config.
// Options can override the exit function or other defaults.
func New(l *slog.Logger, help HelpConfig, opts ...Option) ErrorHandler {
	h := &StandardErrorHandler{
		Logger: l,
		Help:   help,
		Exit:   os.Exit,
	}
	for _, opt := range opts {
		opt(h)
	}

	return h
}

// NewErrNotImplemented creates an unimplemented error with an optional issue tracker link.
func NewErrNotImplemented(issueURL string) error {
	return errors.UnimplementedError(
		errors.IssueLink{IssueURL: issueURL},
		"command not yet implemented",
	)
}

func (h *StandardErrorHandler) Check(err error, prefix string, level string) {
	if err == nil {
		return
	}

	if h.handleSpecialErrors(err) {
		// The special-error presentation (usage output / warn line) has now been
		// emitted. A fatal-level report must still terminate the process: usage
		// and not-yet-implemented errors exit with ExitCodeUsage (2) — the
		// conventional Unix "command misuse" code, distinct from the generic
		// failure code (1) — so a script can tell an invalid invocation from an
		// ordinary runtime failure. Non-fatal levels remain non-exiting.
		if isFatalLevel(level) {
			h.Exit(ExitCodeUsage)
		}

		return
	}

	h.logError(err, prefix, level)
}

// isFatalLevel reports whether level terminates the process. Both the loud and
// quiet fatal levels exit; every other level only logs.
func isFatalLevel(level string) bool {
	return level == LevelFatal || level == LevelFatalQuiet
}

func (h *StandardErrorHandler) handleSpecialErrors(err error) bool {
	if errors.Is(err, ErrNotImplemented) || errors.HasUnimplementedError(err) {
		h.Logger.Warn("Command not yet implemented")

		if links := errors.GetAllIssueLinks(err); len(links) > 0 {
			h.Logger.Info("Track progress", "url", links[0].IssueURL)
		}

		return true
	}

	if errors.Is(err, ErrRunSubCommand) {
		// Usage is printed through the caller-supplied seam, so this module
		// carries no CLI-framework dependency. See SetUsage.
		if h.Usage != nil {
			_ = h.Usage()
		}

		h.Logger.Warn("Subcommand required")

		return true
	}

	if errors.HasAssertionFailure(err) {
		h.Logger.Error("Internal error (assertion failure)", "error", err)

		if h.Logger.Enabled(context.Background(), slog.LevelDebug) {
			h.Logger.Debug("Assertion detail", KeyStacktrace, fmt.Sprintf("%+v", err))
		}

		return false
	}

	return false
}

func (h *StandardErrorHandler) buildLogKVPairs(err error) []any {
	kvPairs := []any{}
	isDebug := h.Logger.Enabled(context.Background(), slog.LevelDebug)

	if isDebug {
		kvPairs = append(kvPairs, KeyStacktrace, extractStackTrace(err))
	}

	if hints := errors.FlattenHints(err); hints != "" {
		kvPairs = append(kvPairs, KeyHints, hints)
	}

	if isDebug {
		if details := errors.FlattenDetails(err); details != "" {
			kvPairs = append(kvPairs, KeyDetails, details)
		}
	}

	if h.Help != nil {
		if msg := h.Help.SupportMessage(); msg != "" {
			kvPairs = append(kvPairs, KeyHelp, msg)
		}
	}

	return kvPairs
}

func (h *StandardErrorHandler) logError(err error, prefix, level string) {
	l := h.Logger
	if len(prefix) > 0 {
		l = l.With("prefix", prefix)
	}

	kvPairs := h.buildLogKVPairs(err)

	switch level {
	case LevelFatal:
		l.Error(err.Error(), kvPairs...)
		h.Exit(ExitCode(err))
	case LevelFatalQuiet:
		l.Debug(err.Error(), kvPairs...)
		h.Exit(ExitCode(err))
	case LevelError:
		l.Error(err.Error(), kvPairs...)
	case LevelWarn:
		l.Warn(err.Error(), kvPairs...)
	default:
		// An unrecognised level must never silently swallow the error;
		// fall back to logging at Error so the failure is still surfaced.
		l.Error(err.Error(), kvPairs...)
	}
}

// Fatal reports err at the fatal level and terminates the process via the
// configured exit function. An ordinary error exits with the code attached via
// WithExitCode (defaulting to 1). A special error — one wrapping ErrRunSubCommand
// or an unimplemented error — still prints its usage/notice presentation and
// then exits with ExitCodeUsage (2), so an invalid CLI invocation is not
// mistaken for success by a calling script. A nil err is a no-op.
func (h *StandardErrorHandler) Fatal(err error, prefixes ...string) {
	h.Check(err, handlePrefix(prefixes...), LevelFatal)
}

func (h *StandardErrorHandler) Error(err error, prefixes ...string) {
	h.Check(err, handlePrefix(prefixes...), LevelError)
}

func (h *StandardErrorHandler) Warn(err error, prefixes ...string) {
	h.Check(err, handlePrefix(prefixes...), LevelWarn)
}

func (h *StandardErrorHandler) SetUsage(usage func() error) {
	h.Usage = usage
}

func handlePrefix(prefixes ...string) string {
	var prefix strings.Builder

	for _, p := range prefixes {
		prefix.WriteString(p)
	}

	return prefix.String()
}
