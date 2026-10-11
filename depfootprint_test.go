package errorhandling_test

import (
	"os/exec"
	"strings"
	"testing"
)

// selfPackage must appear in the runtime scan. Its absence means go list
// reported nothing, which would otherwise pass as a clean graph.
const selfPackage = "gitlab.com/phpboyscout/go/errorhandling"

// runtimeForbidden is what the shipped package may not pull in. aws-sdk-go-v2 and
// azure-sdk-for-go are spelled out because a boundary match on the old bare
// prefixes would miss them.
var runtimeForbidden = []string{
	"gitlab.com/phpboyscout/go-tool-base",
	"github.com/spf13/cobra",
	"github.com/spf13/viper",
	"github.com/spf13/pflag",
	"github.com/charmbracelet",
	"charm.land",
	"go.opentelemetry.io",
	"github.com/aws/aws-sdk-go",
	"github.com/aws/aws-sdk-go-v2",
	"cloud.google.com/go",
	"github.com/Azure/azure-sdk-for-go",
}

// TestDependencyFootprint is the enforceable statement of "framework-free".
// This module formats and routes errors for CLIs, so the temptation to reach for
// a CLI framework is real — and it used to: `Check` once took a variadic
// *cobra.Command purely to print usage. That parameter was removed in favour of
// the SetUsage seam, and Cobra is listed below so it cannot creep back. A caller
// supplies its own usage printer; this module never imports one.
func TestDependencyFootprint(t *testing.T) {
	t.Parallel()

	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}

	sawSelf := false

	for dep := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if dep == selfPackage {
			sawSelf = true
		}

		if bad, ok := forbiddenAtRuntime(dep); ok {
			t.Errorf("forbidden dependency in graph: %s (matched %q)", dep, bad)
		}
	}

	if !sawSelf {
		t.Fatalf("the runtime scan never listed %s, so it did not walk this package", selfPackage)
	}
}

// TestRuntimeMatcherRespectsPathBoundary is the runtime matcher's self-test: a
// healthy package matches nothing, so a matcher that stopped matching would look
// clean.
func TestRuntimeMatcherRespectsPathBoundary(t *testing.T) {
	t.Parallel()

	for _, pkg := range []string{
		"gitlab.com/phpboyscout/go-tool-base/x",
		"github.com/spf13/cobra/x",
		"github.com/spf13/viper/x",
		"github.com/spf13/pflag/x",
		"github.com/charmbracelet/x",
		"charm.land/x",
		"go.opentelemetry.io/x",
		"github.com/aws/aws-sdk-go/x",
		"github.com/aws/aws-sdk-go-v2/x",
		"cloud.google.com/go/x",
		"github.com/Azure/azure-sdk-for-go/x",
		"github.com/aws/aws-sdk-go-v2/service/s3",
		"github.com/Azure/azure-sdk-for-go/sdk/azcore",
	} {
		if _, ok := forbiddenAtRuntime(pkg); !ok {
			t.Errorf("%s should be forbidden at runtime", pkg)
		}
	}

	for _, pkg := range []string{
		"github.com/spf13/viper-fork",
		"gitlab.com/phpboyscout/go/errors",
	} {
		if _, ok := forbiddenAtRuntime(pkg); ok {
			t.Errorf("%s should be permitted at runtime", pkg)
		}
	}
}

func forbiddenAtRuntime(pkg string) (string, bool) {
	for _, bad := range runtimeForbidden {
		if pkg == bad || strings.HasPrefix(pkg, bad+"/") {
			return bad, true
		}
	}

	return "", false
}

// testGraphForbidden is what the repository may not touch at all, test files
// included. It is narrower than the runtime list on purpose: tests may exercise
// what the runtime keeps out, and godog brings pflag, testcontainers brings
// OpenTelemetry.
var testGraphForbidden = []string{
	"gitlab.com/phpboyscout/go-tool-base",
	"github.com/spf13/viper",
	"github.com/spf13/cobra",
	// The estate's one Discord SDK belongs to comms-discord.
	"github.com/disgoorg",
}

// TestTestGraphFootprint covers what the runtime scan cannot see: a module
// imported only by a _test.go file still becomes a direct requirement in go.mod.
func TestTestGraphFootprint(t *testing.T) {
	t.Parallel()

	out, err := exec.Command("go", "list", "-deps", "-test",
		"-f", "{{.ImportPath}}\t{{if .Module}}{{.Module.Path}}{{end}}", "./...").Output()
	if err != nil {
		t.Fatalf("go list -deps -test: %v", err)
	}

	sawTestMain := false
	reported := map[string]bool{}

	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		pkg, module, _ := strings.Cut(line, "\t")

		// Synthetic .test packages exist only under -test, so one proves the
		// scan walked the test graph rather than the build graph.
		if strings.HasSuffix(pkg, ".test") {
			sawTestMain = true
		}

		if !forbiddenInTests(module) || reported[module] {
			continue
		}

		reported[module] = true

		t.Errorf("forbidden module in the test graph: %s (package %s)", module, pkg)
	}

	if !sawTestMain {
		t.Fatal("the scan contains no .test package, so it did not see the test files")
	}
}

// TestTestGraphMatcherRespectsPathBoundary is the guard's self-test: a healthy
// repository matches nothing, so a matcher that stopped matching would look clean.
func TestTestGraphMatcherRespectsPathBoundary(t *testing.T) {
	t.Parallel()

	for _, module := range []string{
		"gitlab.com/phpboyscout/go-tool-base",
		"github.com/spf13/viper",
		"github.com/spf13/cobra",
		"github.com/disgoorg",
		"gitlab.com/phpboyscout/go-tool-base/pkg",
	} {
		if !forbiddenInTests(module) {
			t.Errorf("%s should be forbidden in the test graph", module)
		}
	}

	for _, module := range []string{
		"github.com/spf13/viper-fork",
		"github.com/stretchr/testify",
	} {
		if forbiddenInTests(module) {
			t.Errorf("%s should be permitted in the test graph", module)
		}
	}
}

func forbiddenInTests(module string) bool {
	for _, bad := range testGraphForbidden {
		if module == bad || strings.HasPrefix(module, bad+"/") {
			return true
		}
	}

	return false
}
