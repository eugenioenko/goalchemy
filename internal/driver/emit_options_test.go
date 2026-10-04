package driver

import (
	"sync"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/diagnostics"
	"github.com/eugenioenko/goalchemy/internal/ir"
)

func TestEmissionNamingOptionsAreIndependent(t *testing.T) {
	const target = "test-naming-options"
	var mu sync.Mutex
	observed := map[string]bool{}
	original := &ir.Program{CompactNames: true}
	res := &Result{IR: original}
	Register(target, func(actual *Result, out string) []diagnostics.Diagnostic {
		if actual == res || actual.IR == original {
			t.Error("emitter received shared program header")
		}
		mu.Lock()
		observed[out] = actual.IR.CompactNames
		mu.Unlock()
		return nil
	})
	t.Cleanup(func() { delete(emitters, target) })
	var wg sync.WaitGroup
	for out, compact := range map[string]bool{"readable": false, "compact": true} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ds := EmitWithOptions(target, res, out, EmitOptions{CompactNames: compact}); len(ds) != 0 {
				t.Errorf("emission diagnostics: %v", ds)
			}
		}()
	}
	wg.Wait()
	if ds := Emit(target, res, "default"); len(ds) != 0 {
		t.Fatalf("default emission diagnostics: %v", ds)
	}
	if observed["readable"] || !observed["compact"] || observed["default"] {
		t.Fatalf("wrong naming policies: %v", observed)
	}
	if res.IR != original || !original.CompactNames {
		t.Fatal("emission changed shared program")
	}
}
