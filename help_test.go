package errorhandling

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubHelp is a minimal HelpConfig for exercising the handler's help plumbing.
// The module ships only the interface — concrete channel implementations
// (Slack, Teams, an on-call rota, a wiki URL, …) belong to the consuming
// application, and their own formatting is tested there.
type stubHelp struct {
	team    string
	channel string
}

func (s stubHelp) SupportMessage() string {
	if s.team == "" || s.channel == "" {
		return ""
	}

	return fmt.Sprintf("For assistance, contact %s via %s", s.team, s.channel)
}

func TestHelpConfig_EmptyMessageIsValid(t *testing.T) {
	t.Parallel()

	// Returning "" is the documented way for an implementation to stay silent
	// when it has nothing useful to say (e.g. not configured yet).
	assert.Empty(t, stubHelp{}.SupportMessage())
	assert.Empty(t, stubHelp{team: "Engineering"}.SupportMessage())
	assert.Empty(t, stubHelp{channel: "#support"}.SupportMessage())
	assert.Equal(t, "For assistance, contact Engineering via #support",
		stubHelp{team: "Engineering", channel: "#support"}.SupportMessage())
}

func TestErrorHandler_HelpMessage_InOutput(t *testing.T) {
	log := NewCaptureLogger()

	h := New(log.Logger(), stubHelp{team: "Platform", channel: "#alerts"})
	h.Error(context.Background(), errors.New("something went wrong"))

	entries := log.Entries()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Keyvals, KeyHelp)
	assert.Contains(t, entries[0].Keyvals, "For assistance, contact Platform via #alerts")
}

func TestErrorHandler_NilHelp_NoHelpInOutput(t *testing.T) {
	log := NewCaptureLogger()

	h := New(log.Logger(), nil)
	h.Error(context.Background(), errors.New("something went wrong"))

	entries := log.Entries()
	require.NotEmpty(t, entries)
	assert.NotContains(t, entries[0].Keyvals, KeyHelp)
}

func TestErrorHandler_EmptyHelpMessage_NoHelpInOutput(t *testing.T) {
	log := NewCaptureLogger()

	// A configured-but-silent implementation must not add an empty help key.
	h := New(log.Logger(), stubHelp{})
	h.Error(context.Background(), errors.New("something went wrong"))

	entries := log.Entries()
	require.NotEmpty(t, entries)
	assert.NotContains(t, entries[0].Keyvals, KeyHelp)
}
