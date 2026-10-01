package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTreeHashUsesFileNamesAndContents(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a", "same")
	a, err := treeHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	write("a", "different")
	b, err := treeHash(dir)
	if err != nil || a == b {
		t.Fatalf("content change not detected: %v", err)
	}
	if err := os.Rename(filepath.Join(dir, "a"), filepath.Join(dir, "b")); err != nil {
		t.Fatal(err)
	}
	c, err := treeHash(dir)
	if err != nil || b == c {
		t.Fatalf("path change not detected: %v", err)
	}
}

func TestBDWGCVersion(t *testing.T) {
	version, err := parseBDWGCVersion([]byte("#define GC_TMP_VERSION_MAJOR 8\n#define GC_TMP_VERSION_MINOR 2\n#define GC_TMP_VERSION_MICRO 8 /* 8.2.8 */\n"))
	if err != nil || version != "8.2.8" {
		t.Fatalf("got %q, %v", version, err)
	}
	if _, err := parseBDWGCVersion([]byte("#define GC_TMP_VERSION_MAJOR 8\n")); err == nil {
		t.Fatal("incomplete header accepted")
	}
}

func TestFuzzCommandSmoke(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	path := filepath.Join(t.TempDir(), "report.json")
	cmd := exec.CommandContext(ctx, "go", "run", ".", "-mode", "fuzz", "-root", "../..",
		"-target", "go", "-cases", "1", "-ops", "8", "-budget", "1m", "-out", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fuzz command: %v\n%s", err, output)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var r report
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	if len(r.Results) != 1 || r.Results[0].Status != "pass" || r.Results[0].Cases != 1 || r.ToolchainsLockSHA256 == "" {
		t.Fatalf("unexpected smoke report: %+v", r)
	}
}

func TestUpgradeComparison(t *testing.T) {
	old := report{Mode: "compat", Compiler: "0.1", Toolchains: map[string]string{"go": "old"}, Results: []result{
		{Target: "go", TargetVersion: "1", Status: "pass", Cases: 2, ContractCases: 10, BuildSHA256: "a"},
	}}
	now := report{Mode: "compat", Compiler: "0.2", Toolchains: map[string]string{"go": "new"}, Results: []result{
		{Target: "go", TargetVersion: "2", Status: "fail", Cases: 2, ContractCases: 9, BuildSHA256: "b"},
	}}
	got := strings.Join(changes(old, now), "\n")
	for _, want := range []string{"compiler/profile", "go: old -> new", "go: pass/2 cases/10 contracts -> fail/2 cases/9 contracts", "target version: 1 -> 2", "generated build hash changed"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
