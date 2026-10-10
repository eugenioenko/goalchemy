package os

import (
	stdos "os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadWrite(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "data.txt")
	if s := WriteFile(name, []byte("hello"), 0o640); s != OK {
		t.Fatalf("write %d", s)
	}
	if st, err := stdos.Stat(name); err != nil || st.Mode().Perm()&^0o640 != 0 {
		t.Fatalf("mode %v %v", st, err)
	}
	if s := WriteFile(name, []byte("hi"), 0o600); s != OK {
		t.Fatalf("overwrite %d", s)
	}
	data, s := ReadFile(name)
	if s != OK || string(data) != "hi" {
		t.Fatalf("read %q %d", data, s)
	}
	if st, _ := stdos.Stat(name); st.Mode().Perm()&0o040 == 0 && st.Mode().Perm()&^0o640 != 0 {
		t.Fatalf("overwrite changed mode %v", st.Mode())
	}
	empty := filepath.Join(dir, "empty")
	if s := WriteFile(empty, nil, 0o600); s != OK {
		t.Fatalf("empty write %d", s)
	}
	if data, s := ReadFile(empty); s != OK || data == nil || len(data) != 0 {
		t.Fatalf("empty read %v %d", data, s)
	}
}

func TestStatuses(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f")
	if s := WriteFile(file, []byte("x"), 0o600); s != OK {
		t.Fatal(s)
	}
	for _, c := range []struct {
		name string
		want int
	}{
		{"", NotExist},
		{filepath.Join(dir, "missing"), NotExist},
		{filepath.Join(dir, "missing", "x"), NotExist},
		{dir, IsDir},
		{filepath.Join(file, "x"), NotDir},
		{"a\x00b", Invalid},
	} {
		if data, s := ReadFile(c.name); s != c.want || data != nil {
			t.Errorf("ReadFile(%q) = %v, %d; want %d", c.name, data, s, c.want)
		}
	}
	for _, c := range []struct {
		name string
		want int
	}{
		{"", NotExist},
		{filepath.Join(dir, "missing", "x"), NotExist},
		{dir, IsDir},
		{filepath.Join(file, "x"), NotDir},
		{"a\x00b", Invalid},
	} {
		if s := WriteFile(c.name, []byte("y"), 0o600); s != c.want {
			t.Errorf("WriteFile(%q) = %d; want %d", c.name, s, c.want)
		}
	}
	if stdos.Geteuid() != 0 {
		locked := filepath.Join(dir, "locked")
		if s := WriteFile(locked, []byte("z"), 0); s != OK {
			t.Fatal(s)
		}
		if _, s := ReadFile(locked); s != Permission {
			t.Errorf("locked read %d", s)
		}
		if s := WriteFile(locked, []byte("z"), 0o600); s != Permission {
			t.Errorf("locked write %d", s)
		}
	}
	if s := WriteFile(filepath.Join(dir, "big"), make([]byte, MaxFileBytes+1), 0o600); s != TooLarge {
		t.Errorf("big write %d", s)
	}
	if strings.Contains(file, "\x00") {
		t.Fatal("temp dir")
	}
}
