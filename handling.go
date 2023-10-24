package errorhandling

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/go-errors/errors"
)

func Check(err error, prefix string, level string) {
	if err == nil {
		return
	}

	var stacktrace string
	var es *errors.Error
	if errors.As(err, &es) {
		stacktrace = es.ErrorStack()
	}

	l := log.New(os.Stderr)
	if len(prefix) > 0 {
		l = log.NewWithOptions(os.Stderr, log.Options{
			Prefix: prefix,
		})
	}

	switch level {
	case "fatal":
		l.Fatal(err, "stacktrace", stacktrace)
	case "error":
		l.Error(err, "stacktrace", stacktrace)
	case "warn":
		l.Warn(err, "stacktrace", stacktrace)
	}
}

func CheckFatal(err error, prefixes ...string) {
	Check(err, handlePrefix(prefixes...), "fatal")
}

func CheckError(err error, prefixes ...string) {
	Check(err, handlePrefix(prefixes...), "error")
}

func CheckWarn(err error, prefixes ...string) {
	Check(err, handlePrefix(prefixes...), "warn")
}

func handlePrefix(prefixes ...string) string {
	var prefix string
	prefix = ""
	for _, p := range prefixes {
		prefix += p
	}
	return prefix
}
