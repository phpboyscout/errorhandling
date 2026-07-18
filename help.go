package errorhandling

// HelpConfig supplies contextual support information to attach to reported
// errors — typically "contact <team> via <channel>" for whatever support channel
// your organisation uses.
//
// It is deliberately the only help-related type in this module: the extension
// point, with no opinion about where a team's support channel lives. Slack,
// Teams, an on-call rota, a wiki URL, or a message assembled from configuration
// are all just implementations you supply.
//
// Returning an empty string suppresses the help output entirely, so an
// implementation can stay silent when it has nothing useful to say (for example
// when its channel is not configured yet):
//
//	type slackHelp struct{ team, channel string }
//
//	func (s slackHelp) SupportMessage() string {
//		if s.team == "" || s.channel == "" {
//			return "" // not configured — say nothing
//		}
//
//		return fmt.Sprintf("For assistance, contact %s via Slack channel %s", s.team, s.channel)
//	}
//
// Pass an implementation to [New]; passing nil is valid and disables help
// output.
type HelpConfig interface {
	SupportMessage() string
}
