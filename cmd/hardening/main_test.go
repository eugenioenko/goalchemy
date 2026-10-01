package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
