package errorhandling

import (
	"strings"
	"sync"
	"testing"

	"log/slog"

	"gitlab.com/phpboyscout/go/errors"
)

// A chain across three notional packages, so the reporting site and the failing
// site are different functions.
func failingCall() error { return errors.New("permission denied") }

func middleCall() error {
	return errors.Wrapf(failingCall(), "reading %s", "/etc/config")
}

func surfacingCall() error {
	return errors.Wrap(middleCall(), "loading configuration")
}

// stackAttr returns the reported stacktrace, and whether one was reported.
func stackAttr(t *testing.T, entries []Entry) (string, bool) {
	t.Helper()

	for _, e := range entries {
		for i := 0; i+1 < len(e.Keyvals); i += 2 {
			if key, ok := e.Keyvals[i].(string); ok && key == KeyStacktrace {
				s, _ := e.Keyvals[i+1].(string)

				return s, true
			}
		}
	}

	return "", false
}

func reportAt(t *testing.T, level slog.Level, err error, opts ...ReportOption) []Entry {
	t.Helper()

	capture := NewCaptureLogger()
	capture.SetLevel(level)

	New(capture.Logger(), nil).Fatal(t.Context(), err, opts...)

	return capture.Entries()
}

// The headline of spec 0003: a wrapped error must report where it failed, not
// where it was reported.
func TestReportsTheFailingSiteNotTheWrapSite(t *testing.T) {
	t.Parallel()

	trace, ok := stackAttr(t, reportAt(t, slog.LevelDebug, surfacingCall()))
	if !ok {
		t.Fatal("debug must carry a stacktrace")
	}

	if !strings.Contains(trace, "failingCall") {
		t.Errorf("the failing site must appear, got:\n%s", trace)
	}

	// Everything StackOf would have reported is still there: the origin stack
	// is a superset, not a different view.
	if !strings.Contains(trace, "surfacingCall") {
		t.Errorf("the surfacing site must still appear, got:\n%s", trace)
	}
}

// An unwrapped error has one capture, so nothing about it changes.
func TestUnwrappedErrorReportsWhatItAlwaysDid(t *testing.T) {
	t.Parallel()

	trace, ok := stackAttr(t, reportAt(t, slog.LevelDebug, errors.New("boom")))
	if !ok {
		t.Fatal("debug must carry a stacktrace")
	}

	if !strings.Contains(trace, "TestUnwrappedErrorReportsWhatItAlwaysDid") {
		t.Errorf("want the capture site, got:\n%s", trace)
	}
}

func countFrames(trace string) int {
	n := 0

	for _, line := range strings.Split(strings.TrimRight(trace, "\n"), "\n") {
		if line != "" && !strings.HasPrefix(line, "\t") {
			n++
		}
	}

	return n
}

// The default bound, and that it keeps the innermost frames.
func TestDefaultDepthBoundsTheStack(t *testing.T) {
	t.Parallel()

	trace, ok := stackAttr(t, reportAt(t, slog.LevelDebug, deepError(30)))
	if !ok {
		t.Fatal("debug must carry a stacktrace")
	}

	if got := countFrames(trace); got > DefaultStackDepth {
		t.Errorf("want at most %d frames by default, got %d", DefaultStackDepth, got)
	}

	if !strings.Contains(trace, "deepestFrame") {
		t.Errorf("the innermost frames must be the ones kept, got:\n%s", trace)
	}
}

func TestWithStackDepthRaisesAndRemovesTheBound(t *testing.T) {
	t.Parallel()

	shallow, _ := stackAttr(t, reportAt(t, slog.LevelDebug, deepError(30), WithStackDepth(3)))
	if got := countFrames(shallow); got != 3 {
		t.Errorf("want exactly 3 frames, got %d:\n%s", got, shallow)
	}

	unbounded, _ := stackAttr(t, reportAt(t, slog.LevelDebug, deepError(30), WithStackDepth(-1)))
	if countFrames(unbounded) <= DefaultStackDepth {
		t.Errorf("a negative depth must remove the bound, got %d frames", countFrames(unbounded))
	}
}

// A branched history reports every branch, which is the case DistinctStacks
// exists for and the one OriginStack alone would have discarded.
func TestBranchedHistoryReportsEveryBranch(t *testing.T) {
	t.Parallel()

	inner := failingCall()

	var (
		wrapped error
		wg      sync.WaitGroup
	)

	wg.Add(1)

	go func() {
		defer wg.Done()

		wrapped = errors.Wrap(inner, "wrapped in another goroutine")
	}()
	wg.Wait()

	trace, ok := stackAttr(t, reportAt(t, slog.LevelDebug, wrapped))
	if !ok {
		t.Fatal("debug must carry a stacktrace")
	}

	if !strings.Contains(trace, stackSeparator) {
		t.Errorf("disjoint histories must both be reported, got:\n%s", trace)
	}
}

// Unchanged from 0002 D2: nothing above debug carries a stack.
func TestNoStackAboveDebug(t *testing.T) {
	t.Parallel()

	if _, ok := stackAttr(t, reportAt(t, slog.LevelInfo, surfacingCall())); ok {
		t.Error("info must not carry a stacktrace")
	}
}

// deepError builds a call chain deep enough to exceed the default bound.
func deepError(depth int) error {
	if depth <= 0 {
		return deepestFrame()
	}

	return deepError(depth - 1)
}

func deepestFrame() error { return errors.New("deep failure") }
