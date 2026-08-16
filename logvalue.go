package errorhandling

import (
	"log/slog"

	"gitlab.com/phpboyscout/go/errors"
)

// logValue renders an error as the same structured group go/errors produces, so
// that this module's own wrappers stay transparent at the log boundary.
//
// slog asks the OUTERMOST value whether it implements slog.LogValuer and gives
// up if it does not — it does not unwrap. So a wrapper here without a LogValue
// collapses the whole record to a flat string, losing the hint, the kind, the
// details and every attribute. The documented idiom puts one of ours outermost
// (`return WithExitCode(err, 3)`), which made the common case the broken one.
//
// It is built from go/errors' exported accessors rather than shared with that
// package, whose equivalent is unexported. Duplicating ten lines is the smaller
// cost: exporting it there would put the group's shape in two modules' public
// API, and the shape is that package's to change.
func logValue(err error) slog.Value {
	attrs := []slog.Attr{slog.String("msg", err.Error())}

	if kind := errors.KindOf(err); kind != "" {
		attrs = append(attrs, slog.String("kind", kind))
	}

	if hints := errors.Hints(err); len(hints) > 0 {
		attrs = append(attrs, slog.Any("hint", hints))
	}

	if details := errors.Details(err); len(details) > 0 {
		attrs = append(attrs, slog.Any("detail", details))
	}

	return slog.GroupValue(append(attrs, errors.Attrs(err)...)...)
}
