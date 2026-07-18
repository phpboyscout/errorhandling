package errorhandling_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestDependencyFootprint is the enforceable statement of "framework-free".
// This module formats and routes errors for CLIs, so the temptation to reach for
// a CLI framework is real — and it used to: `Check` once took a variadic
// *cobra.Command purely to print usage. That parameter was removed in favour of
// the SetUsage seam, and Cobra is listed below so it cannot creep back. A caller
// supplies its own usage printer; this module never imports one.
func TestDependencyFootprint(t *testing.T) {
	t.Parallel()

	out, err := exec.Command("go", "list", "-deps", "./...").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}

	forbidden := []string{
		"gitlab.com/phpboyscout/go-tool-base",
		"github.com/spf13/cobra",
		"github.com/spf13/viper",
		"github.com/spf13/pflag",
		"github.com/charmbracelet",
		"charm.land",
		"go.opentelemetry.io",
		"github.com/aws/aws-sdk-go",
		"cloud.google.com/go",
		"github.com/Azure/azure-sdk",
	}

	for _, dep := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		for _, bad := range forbidden {
			if strings.HasPrefix(dep, bad) {
				t.Errorf("forbidden dependency in graph: %s (matched %q)", dep, bad)
			}
		}
	}
}
