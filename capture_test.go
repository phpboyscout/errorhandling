package errorhandling

import (
	"context"
	"log/slog"
	"strings"
	"sync"
)

// CaptureLogger is a test-only slog capture sink. go-tool-base has an
// equivalent in its logger package, but test helpers cannot cross a module
// boundary, so the module vendors its own minimal version: enough to assert on
// the messages and key/value pairs the handler emits, and to raise the level so
// debug-gated output (stack traces, details) can be exercised.
type CaptureLogger struct {
	mu      sync.Mutex
	records []slog.Record
	level   *slog.LevelVar
	logger  *slog.Logger
}

// Entry is a flattened view of a captured record.
type Entry struct {
	Level   slog.Level
	Message string
	Keyvals []any
}

func NewCaptureLogger() *CaptureLogger {
	c := &CaptureLogger{level: new(slog.LevelVar)}
	// Capture everything by default — a test sink that silently dropped debug
	// records would hide exactly the debug-gated output (stack traces, details,
	// the quiet-fatal notice) these tests exist to assert on. Tests that care
	// about level filtering raise the threshold explicitly with SetLevel.
	c.level.Set(slog.LevelDebug)
	c.logger = slog.New(&captureHandler{owner: c, minLevel: c.level})

	return c
}

// Logger returns the *slog.Logger to hand to the code under test.
func (c *CaptureLogger) Logger() *slog.Logger { return c.logger }

// SetLevel raises or lowers the capture threshold.
func (c *CaptureLogger) SetLevel(l slog.Level) { c.level.Set(l) }

// Entries returns the captured records, in order, flattened to message +
// key/value pairs.
func (c *CaptureLogger) Entries() []Entry {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]Entry, len(c.records))

	for i, r := range c.records {
		e := Entry{Level: r.Level, Message: r.Message}

		r.Attrs(func(a slog.Attr) bool {
			e.Keyvals = append(e.Keyvals, a.Key, a.Value.Any())

			return true
		})

		out[i] = e
	}

	return out
}

// Contains reports whether any captured message contains substr.
func (c *CaptureLogger) Contains(substr string) bool {
	for _, e := range c.Entries() {
		if strings.Contains(e.Message, substr) {
			return true
		}
	}

	return false
}

func (c *CaptureLogger) append(r slog.Record) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.records = append(c.records, r)
}

type captureHandler struct {
	owner    *CaptureLogger
	minLevel *slog.LevelVar
	attrs    []slog.Attr
}

func (h *captureHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.minLevel.Level()
}

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	// Fold in any attrs accumulated via WithAttrs so `logger.With(...)` output
	// (the handler's "prefix" key) is visible to assertions.
	if len(h.attrs) > 0 {
		clone := r.Clone()
		clone.AddAttrs(h.attrs...)
		r = clone
	}

	h.owner.append(r)

	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := &captureHandler{owner: h.owner, minLevel: h.minLevel}
	next.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)

	return next
}

func (h *captureHandler) WithGroup(string) slog.Handler { return h }
