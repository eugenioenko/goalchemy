package conformance

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/contracts"
)

func TestHarnessContextStopsBlockedRead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	target := &contracts.Target{Target: "blocked", Harness: &contracts.HarnessConfig{
		Command: []string{"sh", "-c", "exec sleep 5"},
	}}
	h, err := StartContext(ctx, ".", target)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err = h.call(request{Case: "blocked"})
	_ = h.Close()
	if err == nil || !strings.Contains(err.Error(), "process failure") {
		t.Fatalf("expected process failure after deadline, got %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("blocked harness did not stop at context deadline")
	}
}
