package language

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/diagnostics"
	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/testutil"
)

// Cooperative fixtures must schedule identically under both pause models,
// including under other scheduler choice seeds.
func TestGoPauseModelsAgree(t *testing.T) {
	pauseModelsAgree(t, "go")
}

func TestTypeScriptPauseModelsAgree(t *testing.T) {
	pauseModelsAgree(t, "typescript")
}

func TestSwiftPauseModelsAgree(t *testing.T) {
	pauseModelsAgree(t, "swift")
}

func TestCSharpPauseModelsAgree(t *testing.T) {
	pauseModelsAgree(t, "csharp")
}

func TestPythonPauseModelsAgree(t *testing.T) {
	pauseModelsAgree(t, "python")
}

func TestRustPauseModelsAgree(t *testing.T) {
	pauseModelsAgree(t, "rust")
}

func TestJavaPauseModelsAgree(t *testing.T) {
	pauseModelsAgree(t, "java")
}

// pauseModelsAgree compiles every cooperative fixture for target under both
// pause models and compares fresh executions under each scheduler seed.
func pauseModelsAgree(t *testing.T, target string) {
	fixtures, err := testutil.Discover("testdata")
	if err != nil {
		t.Fatal(err)
	}
	only := os.Getenv("FIXTURE")
	suite := testutil.NewSuite(t)
	for _, f := range fixtures {
		if f.Gate != "cooperative" || len(f.Reject) > 0 || (only != "" && !strings.Contains(f.Name, only)) {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			t.Parallel()
			work := t.TempDir()
			res, ds := suite.Build(f)
			if diagnostics.HasErrors(ds) {
				t.Fatalf("source build: %v", ds)
			}
			programs := map[string]*testutil.Program{}
			for _, model := range []string{"frames", "native"} {
				out := filepath.Join(work, model)
				if ds := driver.EmitWithOptions(target, res, out, driver.EmitOptions{PauseModel: model}); len(ds) > 0 {
					t.Fatalf("%s: %v", model, ds)
				}
				program, err := testutil.Prepare(target, out)
				if err != nil {
					t.Fatalf("%s build: %v", model, err)
				}
				if program.Reused {
					t.Logf("reused %s %s build", target, model)
				}
				programs[model] = program
			}
			for _, seed := range []string{"1", "7", "4242"} {
				frames, native := runSeed(t, programs["frames"], seed), runSeed(t, programs["native"], seed)
				if frames != native {
					t.Errorf("seed %s: native differs from frames\n=== frames\n%s=== native\n%s", seed, frames, native)
				}
			}
		})
	}
}

func runSeed(t *testing.T, program *testutil.Program, seed string) testutil.Observation {
	t.Helper()
	o, err := program.Run(time.Minute, "GOALCHEMY_SEED="+seed)
	if err != nil {
		t.Fatal(err)
	}
	return o
}
