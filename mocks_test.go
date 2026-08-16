package errorhandling_test

import (
	"gitlab.com/phpboyscout/go/errorhandling"
	"gitlab.com/phpboyscout/go/errorhandling/mocks"
)

// The published mock has to satisfy the interface it mocks. Nothing else in this
// module imports mocks, so without this line a mock generated against an older
// ErrorHandler still compiles here and fails only in a consumer's tests — which
// is how it shipped stale through v0.3.0.
var _ errorhandling.ErrorHandler = (*mocks.MockErrorHandler)(nil)

var _ errorhandling.HelpConfig = (*mocks.MockHelpConfig)(nil)
