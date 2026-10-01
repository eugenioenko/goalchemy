// Package sanitize runs the language fixtures and saved regressions on the
// C target built with clang's address and undefined-behavior sanitizers.
// The Boehm collector scans memory conservatively without tripping either
// sanitizer: its own code is not instrumented, and stack-use-after-return
// detection, which hides frames from it, is off.
package sanitize

import (
	"path/filepath"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/testutil"
)

func TestSanitizedC(t *testing.T) {
	if testing.Short() {
		t.Skip("sanitizer matrix runs in the full suite")
	}
	if _, err := testutil.SanitizerClang(); err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{"../language/testdata", "../corpus"} {
		fixtures, err := testutil.Discover(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range fixtures {
			if len(f.Reject) > 0 {
				continue
			}
			t.Run(filepath.Base(root)+"/"+f.Name, func(t *testing.T) {
				t.Parallel()
				testutil.RunFixture(t, f, []string{"c-sanitize"})
			})
		}
	}
}
