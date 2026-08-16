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

// This module's attachments are meant to be transparent: attaching an exit code
// or an outcome states how the program should end and changes nothing about
// what the error is.
//
// That has to hold at the log boundary too. slog asks the OUTERMOST value
// whether it is a slog.LogValuer, so a wrapper of ours without one collapses the
// whole record to a flat string — losing the hint, the kind, the details and
// every attribute the error was carrying. The documented idiom puts our wrapper
// outermost:
//
//	return errorhandling.WithExitCode(err, 3)
//
// so the common case was the broken one.
func TestOurWrappersKeepTheErrorStructured(t *testing.T) {
	t.Parallel()

	for name, wrap := range map[string]func(error) error{
		"exit code": func(err error) error {
			return errorhandling.WithExitCode(err, 3)
		},
		"outcome": func(err error) error {
			return errorhandling.WithOutcome(err, errorhandling.Outcome{Code: 3})
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			inner := errors.WithAttrs(
				errors.WithDetail(
					errors.WithHint(errors.New("config file not found"), "run init"),
					"stat: no such file"),
				slog.String("path", "/etc/tool.yaml"))

			buf := errorhandling.NewCaptureLogger()
			h := errorhandling.New(buf.Logger(), nil)

			h.Error(context.Background(), wrap(inner))

			group, ok := buf.ErrGroup()
			require.True(t, ok, "our wrapper must not flatten the record to a string")
			assert.Equal(t, "config file not found", group["msg"])
			assert.NotEmpty(t, group["hint"], "the hint survives our wrapper")
			assert.NotEmpty(t, group["detail"], "so do details")
			assert.Equal(t, "/etc/tool.yaml", group["path"], "and attributes")
			assert.NotEmpty(t, group["kind"], "and the error still names its kind")
		})
	}
}
