# Add a support channel

When your tool fails in someone else's hands, the most useful thing you can add is
*where to get help*. `HelpConfig` appends a support message to every reported error.

## Implement the interface

The module ships the interface and **no implementations** — it has no opinion about
where your team's support channel lives:

```go
type HelpConfig interface {
	SupportMessage() string
}
```

So write the one that matches your organisation:

```go
type slackHelp struct {
	Team    string
	Channel string
}

func (s slackHelp) SupportMessage() string {
	if s.Team == "" || s.Channel == "" {
		return "" // not configured — stay silent
	}

	return fmt.Sprintf("For assistance, contact %s via Slack channel %s", s.Team, s.Channel)
}
```

Pass it when constructing the handler:

```go
handler := errorhandling.New(logger, slackHelp{Team: "Platform", Channel: "#support"})
```

Every reported error now carries a `help` field:

```
level=ERROR msg="deploy failed" hints="Check your cluster credentials" help="For assistance, contact Platform via Slack channel #support"
```

## Returning an empty string suppresses it

An implementation that returns `""` adds nothing to the output. That is the designed
way to stay quiet when a channel isn't configured yet — no `nil` juggling and no
half-rendered "contact  via " messages.

Passing `nil` for the whole `HelpConfig` disables help output entirely:

```go
handler := errorhandling.New(logger, nil)
```

## Anything can be a help channel

Because it's a one-method interface, the message can come from anywhere — a wiki URL,
an on-call rota looked up at startup, a value read from configuration:

```go
type configuredHelp struct{ msg string }

func (c configuredHelp) SupportMessage() string { return c.msg }

handler := errorhandling.New(logger, configuredHelp{
	msg: cfg.GetString("support.message"),
})
```

Keep it cheap: `SupportMessage` is called on **every** reported error, so it should not
do I/O. Resolve the value once at startup and hold it.

## Why the module ships no implementations

A Slack or Teams struct is an *opinion* — about which vendor you use, and about the
config shape that populates it. Baking either into the module would make every consumer
carry someone else's choice. The interface is the extension point; the implementation
is yours, exactly as it should be.

## Related

- [The reporting model](../explanation/reporting-model.md#what-gets-rendered-and-when)
- [Test error handling](test-error-handling.md)
