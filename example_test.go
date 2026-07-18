package errorhandling_test

import (
	"fmt"

	"github.com/cockroachdb/errors"

	"gitlab.com/phpboyscout/go/errorhandling"
)

func ExampleWithUserHint() {
	err := errors.New("connection refused")
	hinted := errorhandling.WithUserHint(err, "Check that the server is running and the port is correct")

	fmt.Println(errors.FlattenHints(hinted))
	// Output: Check that the server is running and the port is correct
}

func ExampleWrapWithHint() {
	err := errors.New("file not found")
	wrapped := errorhandling.WrapWithHint(err, "loading config", "Run 'mytool init' to create the config file")

	fmt.Println(wrapped.Error())
	fmt.Println(errors.FlattenHints(wrapped))
	// Output:
	// loading config: file not found
	// Run 'mytool init' to create the config file
}

func ExampleNew() {
	handler := errorhandling.New(nil, nil)
	_ = handler // Use handler.Check, handler.Error, handler.Fatal, handler.Warn
}

// slackHelp is an example HelpConfig implementation. The module ships only the
// interface, so an application defines whatever support channel it actually
// uses — Slack here, but equally Teams, an on-call rota, or a wiki URL.
type slackHelp struct {
	Team    string
	Channel string
}

func (s slackHelp) SupportMessage() string {
	if s.Team == "" || s.Channel == "" {
		return "" // not configured — suppress the help output
	}

	return fmt.Sprintf("For assistance, contact %s via Slack channel %s", s.Team, s.Channel)
}

func ExampleHelpConfig() {
	var help errorhandling.HelpConfig = slackHelp{
		Team:    "mycompany",
		Channel: "#dev-support",
	}

	fmt.Println(help.SupportMessage())
	// Output: For assistance, contact mycompany via Slack channel #dev-support
}
