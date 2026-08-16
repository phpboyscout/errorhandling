package errorhandling_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"

	"gitlab.com/phpboyscout/go/errors"

	"gitlab.com/phpboyscout/go/errorhandling"
)

// A terminal outcome is declared beside the sentinel it describes, so the error
// carries its own disposition instead of the handler switching on it.
func ExampleWithOutcome() {
	errUpdateComplete := errorhandling.WithOutcome(
		errors.NewSentinel("example.update_complete", "update complete — restart required"),
		errorhandling.Outcome{
			// Zero is legitimate: this outcome is terminal AND successful.
			Code:    0,
			Level:   slog.LevelWarn,
			Message: "update complete — please run the command again",
		},
	)

	// The error arrives as a group, so suppressing it here means matching on the
	// group name its attributes are nested under, not on a top-level key.
	quiet := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey || slices.Contains(groups, errorhandling.KeyError) {
				return slog.Attr{}
			}

			return a
		},
	}))

	handler := errorhandling.New(quiet, nil)

	fmt.Println("exit code:", handler.Fatal(context.Background(), errUpdateComplete))
	// Output:
	// level=WARN msg="update complete — please run the command again"
	// exit code: 0
}

// The shape a main function takes: the handler reports and yields a code, and
// main is the only thing that exits.
func ExampleErrorHandler_Fatal() {
	handler := errorhandling.New(slog.Default(), nil)

	run := func() error { return nil }

	if err := run(); err != nil {
		os.Exit(handler.Fatal(context.Background(), err))
	}

	fmt.Println("clean run")
	// Output: clean run
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
