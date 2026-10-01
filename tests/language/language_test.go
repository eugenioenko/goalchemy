// Package language runs the source language fixtures: each program runs
// natively and through every released target, and observations must agree.
package language

import (
	"os"
	"strings"
	"testing"

	"goalchemy/internal/testutil"
)

var targets = []string{"go", "typescript"}

func TestFixtures(t *testing.T) {
	targets := targets
	if only := os.Getenv("GOALCHEMY_TEST_TARGETS"); only != "" {
		targets = strings.Split(only, ",")
	}
	fixtures, err := testutil.Discover("testdata")
	if err != nil {
		t.Fatal(err)
	}
	only := os.Getenv("FIXTURE")
	for _, f := range fixtures {
		if only != "" && !strings.Contains(f.Name, only) {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			t.Parallel()
			testutil.RunFixture(t, f, targets)
		})
	}
}
