package errorhandling

import (
	"context"
	"log/slog"
	"strings"

	"gitlab.com/phpboyscout/go/errors"
)

// Keys this module adds to a log record. Everything the ERROR carries — its
// message, kind, hints, details and attributes — arrives under KeyError via
// slog.LogValuer, so none of that needs a key here.
const (
	// KeyError is the group an error is logged under.
	KeyError = "err"
	// KeyHelp carries the support message from [HelpConfig].
	KeyHelp = "help"
	// KeyPrefix carries the caller-supplied prefix.
	KeyPrefix = "prefix"
	// KeyStacktrace carries the stack, at debug only.
	KeyStacktrace = "stacktrace"
)

var (
	// ErrNotImplemented marks a command that exists but does nothing yet.
	ErrNotImplemented = WithOutcome(
		errors.NewSentinel("errorhandling.not_implemented", "command not yet implemented"),
		Outcome{Code: ExitCodeUsage, Level: slog.LevelWarn},
	)

	// ErrRunSubCommand marks a parent command invoked without a subcommand,
	// where being invoked without one is itself the mistake.
	//
	// That is a choice, not a rule: a parent that only groups its children can
	// equally treat a bare invocation as a request for help and succeed. Return
	// this when the command genuinely cannot act on its own.
	ErrRunSubCommand = WithOutcome(
		errors.NewSentinel("errorhandling.run_subcommand", "subcommand required"),
		Outcome{Code: ExitCodeUsage, Level: slog.LevelWarn, Usage: true},
	)

	// ErrUnknownSubCommand marks a parent command given a verb it does not have.
	//
	// Cobra reports an unknown command for the root only, so a parent that wants
	// to catch a mistyped subcommand has to do it in its own run function. Wrap
	// this with the offending verb and the command path:
	//
	//	errors.Wrapf(errorhandling.ErrUnknownSubCommand,
	//		"unknown command %q for %q", args[0], cmd.CommandPath())
	//
	// Distinct from ErrRunSubCommand: there, no subcommand was given at all;
	// here, one was, and it does not exist.
	ErrUnknownSubCommand = WithOutcome(
		errors.NewSentinel("errorhandling.unknown_subcommand", "unknown subcommand"),
		Outcome{Code: ExitCodeUsage, Level: slog.LevelWarn, Usage: true},
	)

	// ErrAssertionFailure marks a violated internal invariant — a bug in the
	// program rather than a mistake by its user.
	//
	// It used to be reported on a second log line saying so. It no longer needs
	// one: the kind is on the record, which a query can filter on and a string
	// prefix cannot.
	ErrAssertionFailure = errors.NewSentinel(
		"errorhandling.assertion_failure", "internal invariant violated")
)

// ErrorHandler reports an error once, with everything it carries.
//
// # Nothing here exits the process
//
// [ErrorHandler.Fatal] returns the exit code it believes the process should
// use; main decides what to do with it. A library calling os.Exit skips every
// deferred cleanup between itself and main — which cost go-tool-base a
// sync.Once and a manual flush before every fatal path, covering the one
// cleanup somebody noticed.
type ErrorHandler interface {
	// Fatal reports a terminal error and returns the exit code to use. A nil
	// error reports nothing and returns 0.
	Fatal(ctx context.Context, err error, opts ...ReportOption) int

	// Error reports a non-terminal failure.
	Error(ctx context.Context, err error, opts ...ReportOption)

	// Warn reports something worth saying that is not a failure.
	Warn(ctx context.Context, err error, opts ...ReportOption)

	// SetUsage registers the function used to print usage for an error whose
	// [Outcome] asks for it. CLI frameworks supply their own printer — with
	// Cobra that is SetUsage(cmd.Usage), typically per-command in pre-run so
	// the usage shown belongs to the command that actually failed.
	SetUsage(usage func() error)
}

// ReportOption adjusts a single report.
type ReportOption func(*reportConfig)

type reportConfig struct {
	prefix string
	quiet  bool

	// stackDepth bounds the reported stack. Zero means unset, so configure can
	// tell "the caller said nothing" from "the caller asked for no bound",
	// which is what a negative value says.
	stackDepth int
}

// WithPrefix labels the report, for distinguishing which phase of a run failed.
func WithPrefix(prefix string) ReportOption {
	return func(c *reportConfig) { c.prefix = prefix }
}

// Quietly demotes the log line to debug without changing the exit code.
//
// For an expected, user-initiated end — an interrupt, say — where the non-zero
// exit code is the signal and an error line would be noise. The message is
// still emitted at debug, so --debug continues to surface it.
func Quietly() ReportOption {
	return func(c *reportConfig) { c.quiet = true }
}

// WithStackDepth bounds how many frames of a reported stack are shown.
//
// The default is [DefaultStackDepth]. A negative value removes the bound, for a
// caller that wants everything the error captured.
//
// The frames kept are the innermost, so the failure and its immediate callers
// survive and runtime.main and runtime.goexit are the first to go. That is the
// end worth losing.
//
// This bounds REPORTING, not capture. go/errors captures more than this so the
// frames are there when a caller raises the bound; the two answer different
// questions, one about the cost of making an error and one about the width of
// a terminal.
func WithStackDepth(frames int) ReportOption {
	return func(c *reportConfig) { c.stackDepth = frames }
}

// StandardErrorHandler is the default [ErrorHandler].
type StandardErrorHandler struct {
	Logger *slog.Logger
	Help   HelpConfig
	Usage  func() error
}

// New creates an ErrorHandler. A nil help config disables the support message.
func New(l *slog.Logger, help HelpConfig) ErrorHandler {
	return &StandardErrorHandler{Logger: l, Help: help}
}

func (h *StandardErrorHandler) Fatal(ctx context.Context, err error, opts ...ReportOption) int {
	if err == nil {
		return 0
	}

	cfg := configure(opts...)

	level := slog.LevelError
	if cfg.quiet {
		level = slog.LevelDebug
	}

	code := ExitCode(err)

	// An outcome overrides both, because the error knows more about what it
	// means than the call site does.
	if outcome, ok := OutcomeOf(err); ok {
		level = outcome.Level
		code = outcome.Code

		if outcome.Usage && h.Usage != nil {
			_ = h.Usage()
		}
	}

	h.report(ctx, err, level, cfg)

	return code
}

func (h *StandardErrorHandler) Error(ctx context.Context, err error, opts ...ReportOption) {
	h.at(ctx, err, slog.LevelError, opts...)
}

func (h *StandardErrorHandler) Warn(ctx context.Context, err error, opts ...ReportOption) {
	h.at(ctx, err, slog.LevelWarn, opts...)
}

func (h *StandardErrorHandler) SetUsage(usage func() error) { h.Usage = usage }

// at reports a non-terminal error, honouring an outcome's level if it has one.
func (h *StandardErrorHandler) at(
	ctx context.Context, err error, level slog.Level, opts ...ReportOption,
) {
	if err == nil {
		return
	}

	if outcome, ok := OutcomeOf(err); ok {
		level = outcome.Level
	}

	h.report(ctx, err, level, configure(opts...))
}

// report emits exactly one record.
//
// The error is handed to slog rather than taken apart: its message, kind,
// hints, details and attributes arrive as a structured group through
// slog.LogValuer. This module adds only what the PROCESS knows and the error
// cannot — the support message, and the caller's prefix.
func (h *StandardErrorHandler) report(
	ctx context.Context, err error, level slog.Level, cfg reportConfig,
) {
	message := err.Error()
	if outcome, ok := OutcomeOf(err); ok && outcome.Message != "" {
		message = outcome.Message
	}

	attrs := []any{KeyError, err}

	if cfg.prefix != "" {
		attrs = append(attrs, KeyPrefix, cfg.prefix)
	}

	if h.Help != nil {
		if support := h.Help.SupportMessage(); support != "" {
			attrs = append(attrs, KeyHelp, support)
		}
	}

	// The stack is large and rarely what a log line is for, so LogValue omits
	// it. Debug is where someone has asked for everything.
	if h.Logger.Enabled(ctx, slog.LevelDebug) {
		if trace := renderStacks(err, cfg.stackDepth); trace != "" {
			attrs = append(attrs, KeyStacktrace, trace)
		}
	}

	h.Logger.Log(ctx, level, message, attrs...)
}

func configure(opts ...ReportOption) reportConfig {
	var cfg reportConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// NewErrNotImplemented returns an unimplemented-command error carrying a link
// to the issue tracking the work.
//
// The URL is an attribute rather than a bespoke payload type, so it reaches a
// log record and a span without this module doing anything.
func NewErrNotImplemented(issueURL string) error {
	err := errors.WithStack(ErrNotImplemented)
	if issueURL == "" {
		return err
	}

	return errors.WithAttrs(err, slog.String("issue_url", issueURL))
}

// UnknownSubCommand reports a parent command given a verb it does not have,
// naming the verb and the command that rejected it.
//
// It exists so the message and the sentinel do not drift between CLIs. The cobra
// glue that calls it cannot live here — this module deliberately imports no CLI
// framework, and a module holding eight lines of glue would not earn its keep —
// but the half worth sharing is the wording and the identity, not the closure:
//
//	RunE: func(cmd *cobra.Command, args []string) error {
//		if len(args) > 0 {
//			return errorhandling.UnknownSubCommand(args[0], cmd.CommandPath())
//		}
//
//		return cmd.Usage()
//	}
func UnknownSubCommand(verb, path string) error {
	return errors.Wrapf(ErrUnknownSubCommand, "unknown command %q for %q", verb, path)
}

// NewAssertionFailure returns an error denoting a bug in the program.
func NewAssertionFailure(format string, args ...any) error {
	return errors.Wrapf(ErrAssertionFailure, format, args...)
}

// Prefix joins prefix fragments, for a caller assembling one from parts.
func Prefix(parts ...string) string { return strings.Join(parts, "") }
