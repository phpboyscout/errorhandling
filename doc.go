// Package errorhandling reports an error once, with everything it carries, and
// tells the caller what exit code to use.
//
// It builds on gitlab.com/phpboyscout/go/errors: hints, structured attributes,
// stacks and a stable kind travel on the error itself and reach a log record
// through slog.LogValuer. This module adds only what the PROCESS knows and the
// error cannot — the support message from [HelpConfig], and the caller's prefix.
//
// # Nothing here exits
//
// [ErrorHandler.Fatal] returns an exit code; main decides. A library calling
// os.Exit skips every deferred cleanup between itself and main.
//
// # A terminal error carries its own disposition
//
// [Outcome] says how an error ends — its exit code, how loudly to report it,
// whether to print usage — declared beside the sentinel it describes. Zero is a
// legitimate code: an outcome can be terminal and successful.
//
// See spec 0002 on this project's wiki for the decisions and what was rejected.
package errorhandling
