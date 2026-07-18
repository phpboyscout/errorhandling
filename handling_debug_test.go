package errorhandling

import (
	"log/slog"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckDebug(t *testing.T) {
	log := NewCaptureLogger()
	log.SetLevel(slog.LevelDebug)

	h := New(log.Logger(), nil)

	err := errors.New("debug error")
	h.Check(err, "", LevelError)

	entries := log.Entries()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "debug error")
	assert.Contains(t, entries[0].Keyvals, KeyStacktrace)
}

func TestCheckStacktrace(t *testing.T) {
	log := NewCaptureLogger()
	log.SetLevel(slog.LevelInfo)

	h := New(log.Logger(), nil)

	err := errors.New("stacktrace error")
	h.Check(err, "", LevelError)

	entries := log.Entries()
	require.NotEmpty(t, entries)
	assert.NotContains(t, entries[0].Keyvals, KeyStacktrace)
}
