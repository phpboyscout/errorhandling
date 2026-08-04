package errorhandling

import (
	"strings"

	"gitlab.com/phpboyscout/go/errors"
)

// DefaultStackDepth bounds a reported stack when the caller names no bound.
//
// Twenty frames is deep enough to cross several packages and still show where
// the failure happened, and shallow enough that a terminal stays readable. It
// is deliberately smaller than what go/errors captures: capture is bounded so
// that making an error in a hot path is cheap, reporting is bounded so that
// reading one is possible, and those are different questions.
//
// Raise it, or remove the bound entirely, with [WithStackDepth].
const DefaultStackDepth = 20

// stackSeparator divides one stack from the next when an error's history
// branched. The common case has exactly one stack, so it is never seen.
const stackSeparator = "\n\n"

// renderStacks formats the stacks an error carries, bounded to depth frames
// each, or "" when it carries none.
//
// It reports DistinctStacks rather than StackOf. StackOf answers "where did
// this surface", which for a wrapped error is the reporting site rather than
// the failing one: three packages deep it names neither the failing package nor
// the failing line. DistinctStacks answers "how did it get here", and on an
// ordinary chain that is the origin alone, because every outer capture repeats
// a descent already described by a deeper one.
//
// More than one survives only when a history genuinely branched: an error
// stored and wrapped from another call path, or wrapped in another goroutine.
// Then both are worth having, and the reader gets both.
//
// See spec 0003.
func renderStacks(err error, depth int) string {
	stacks := errors.DistinctStacks(err)
	if len(stacks) == 0 {
		return ""
	}

	if depth == 0 {
		depth = DefaultStackDepth
	}

	rendered := make([]string, 0, len(stacks))

	for _, stack := range stacks {
		rendered = append(rendered, boundStack(stack, depth).String())
	}

	return strings.Join(rendered, stackSeparator)
}

// boundStack keeps at most depth frames, from the innermost outwards. A depth
// below zero keeps everything.
//
// The frames are ordered innermost first, so slicing from the front keeps the
// failure and its immediate callers and drops the runtime scaffolding at the
// bottom, which is the end nobody reads.
func boundStack(stack errors.StackTrace, depth int) errors.StackTrace {
	if depth < 0 || len(stack) <= depth {
		return stack
	}

	return stack[:depth]
}
