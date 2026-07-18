// Package errorhandling provides structured, user-friendly error reporting for
// command-line tools.
//
// The [ErrorHandler] interface offers Check (route an error through the
// reporting pipeline), Fatal (report and exit), Error (non-terminating report),
// and Warn. Errors are rendered with any user-facing hints attached via
// cockroachdb/errors (WithHint/WithHintf, or the [WithUserHint] helpers here),
// plus an optional support-channel message supplied through [HelpConfig].
//
// An exit code can be attached to an error value with [WithExitCode] and is
// honoured when the error is reported at Fatal level, so the code that knows
// *why* something failed decides how the process exits.
//
// Stack traces and error details are extracted from cockroachdb/errors and
// emitted only when the logger has debug enabled, giving rich diagnostics on
// demand without cluttering normal output.
//
// The package carries no CLI-framework dependency. Where a parent command needs
// to print usage (an error wrapping [ErrRunSubCommand]), the caller supplies the
// printer through SetUsage — with Cobra, that is cmd.Usage.
package errorhandling
