package os

import (
	"errors"
	"io/fs"
	stdos "os"
	"path/filepath"
	"testing"

	goerrors "github.com/eugenioenko/goalchemy/std/errors"
)

func TestMatchesGo(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f")
	if err := WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	reads := []string{"", filepath.Join(dir, "missing"), filepath.Join(dir, "missing", "x"), dir, filepath.Join(file, "x"), "a\x00b"}
	for _, name := range reads {
		_, got := ReadFile(name)
		_, want := stdos.ReadFile(name)
		compare(t, "ReadFile", name, got, want)
	}
	writes := []string{"", filepath.Join(dir, "missing", "x"), dir, filepath.Join(file, "x"), "a\x00b"}
	for _, name := range writes {
		got := WriteFile(name, []byte("y"), 0o600)
		want := stdos.WriteFile(name, []byte("y"), 0o600)
		compare(t, "WriteFile", name, got, want)
	}
	if stdos.Geteuid() != 0 {
		locked := filepath.Join(dir, "locked")
		if err := WriteFile(locked, nil, 0); err != nil {
			t.Fatal(err)
		}
		_, got := ReadFile(locked)
		_, want := stdos.ReadFile(locked)
		compare(t, "ReadFile", locked, got, want)
	}
	data, err := ReadFile(file)
	if err != nil || string(data) != "x" {
		t.Fatalf("%q %v", data, err)
	}
}

func compare(t *testing.T, op, name string, got, want error) {
	t.Helper()
	if got == nil || want == nil {
		t.Fatalf("%s(%q) = %v, Go %v", op, name, got, want)
	}
	if got.Error() != want.Error() {
		t.Errorf("%s(%q) text %q, Go %q", op, name, got.Error(), want.Error())
	}
	pairs := []struct{ mine, gos error }{{ErrNotExist, fs.ErrNotExist}, {ErrExist, fs.ErrExist}, {ErrPermission, fs.ErrPermission}, {ErrInvalid, fs.ErrInvalid}, {goerrors.ErrUnsupported, errors.ErrUnsupported}}
	for _, p := range pairs {
		if goerrors.Is(got, p.mine) != errors.Is(want, p.gos) {
			t.Errorf("%s(%q): Is(%v) = %v, Go %v", op, name, p.mine, goerrors.Is(got, p.mine), errors.Is(want, p.gos))
		}
	}
	var pe *PathError
	var gpe *fs.PathError
	if !goerrors.As(got, &pe) || !errors.As(want, &gpe) || pe.Op != gpe.Op || pe.Path != gpe.Path {
		t.Errorf("%s(%q): PathError %+v, Go %+v", op, name, pe, gpe)
	}
}
