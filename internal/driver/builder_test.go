package driver_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/diagnostics"
	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/subset"
)

func TestBuilderReloadsSourceAndKeepsOptionsIndependent(t *testing.T) {
	builder, ds := driver.NewBuilder()
	if diagnostics.HasErrors(ds) {
		t.Fatal(ds)
	}
	dir := t.TempDir()
	write := func(name, data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example\n\ngo 1.25\n")
	write("main.go", "package main\nfunc main() { println(1) }\n")
	first, ds := builder.Build(context.Background(), driver.Options{Dir: dir})
	if diagnostics.HasErrors(ds) {
		t.Fatal(ds)
	}
	write("main.go", "package main\nfunc main() { go func() {}() }\n")
	if _, ds := builder.Build(context.Background(), driver.Options{Dir: dir}); !diagnostics.HasErrors(ds) {
		t.Fatal("sequential build reused previous source or accepted a goroutine")
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, ds := builder.Build(context.Background(), driver.Options{Dir: dir, Gate: subset.Cooperative})
			if diagnostics.HasErrors(ds) {
				t.Error(ds)
				return
			}
			if res.IR == first.IR || res.Program == first.Program || res.Registry == first.Registry || res.Catalog != first.Catalog {
				t.Error("builder did not isolate source while sharing the catalog")
			}
			for _, model := range []string{"frames", "native"} {
				if ds := driver.EmitWithOptions("go", res, filepath.Join(t.TempDir(), model), driver.EmitOptions{PauseModel: model}); diagnostics.HasErrors(ds) {
					t.Error(ds)
				}
			}
		}()
	}
	wg.Wait()
	t.Setenv("GOALCHEMY_ROOT", t.TempDir())
	if _, ds := driver.Build(context.Background(), driver.Options{Dir: dir}); !diagnostics.HasErrors(ds) {
		t.Fatal("standalone Build reused the previous catalog despite a new root")
	}
}
