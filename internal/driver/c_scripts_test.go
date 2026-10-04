package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// A generated C library can have no selected runtime operation files. Exercise
// the real archive and native consumer, rather than treating an unmatched glob
// as a source file or silently omitting present runtime sources.
func TestCLibraryBuildWithOptionalRuntimeSources(t *testing.T) {
	for _, runtime := range []bool{false, true} {
		t.Run(fmt.Sprint(runtime), func(t *testing.T) {
			dir := t.TempDir()
			files := map[string]string{
				"main.c":             "extern int support(void); int answer(void) { return support()+40; }\n",
				"rt/types/support.c": "int support(void) { return 2; }\n",
				"build.sh":           cLibBuild,
			}
			consumer := "extern int answer(void); int main(void) { return answer() == 42 ? 0 : 1; }\n"
			if runtime {
				files["rt/runtime/optional.c"] = "int optional(void) { return 3; }\n"
				consumer = "extern int answer(void); extern int optional(void); int main(void) { return answer() == 42 && optional() == 3 ? 0 : 1; }\n"
			}
			files["consumer.c"] = consumer
			for path, source := range files {
				target := filepath.Join(dir, path)
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			run := func(name string, args ...string) {
				cmd := exec.Command(name, args...)
				cmd.Dir = dir
				cmd.Env = ToolEnv()
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("%s: %v\n%s", name, err, output)
				}
			}
			run("sh", "build.sh")
			run("cc", "consumer.c", "libgoalchemy.a", "-o", "consumer")
			run(filepath.Join(dir, "consumer"))
		})
	}
}
