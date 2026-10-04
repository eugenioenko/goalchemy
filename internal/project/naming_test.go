package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompactNamesConfiguration(t *testing.T) {
	for _, tc := range []struct {
		value   string
		want    bool
		invalid bool
	}{
		{"", false, false}, {"compact_names: true\n", true, false},
		{"compact_names: false\n", false, false}, {"compact_names: yes\n", false, true},
	} {
		t.Run(tc.value, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), FileName)
			data := "schema_version: 1\npackages: [.]\n" + tc.value + "targets:\n  go: {out: out/go}\n"
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			c, err := Load(path)
			if tc.invalid {
				if err == nil {
					t.Fatal("accepted non-boolean compact_names")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.CompactNames != tc.want {
				t.Fatalf("compact_names = %v, want %v", c.CompactNames, tc.want)
			}
		})
	}
}
