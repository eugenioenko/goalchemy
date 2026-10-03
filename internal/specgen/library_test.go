package specgen

import (
	"bytes"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/eugenioenko/goalchemy/internal/catalog"
)

type overlayFS struct {
	fs.FS
	over fstest.MapFS
}

func (o overlayFS) Open(name string) (fs.File, error) {
	if _, ok := o.over[name]; ok {
		return o.over.Open(name)
	}
	return o.FS.Open(name)
}

func TestLibraryDoc(t *testing.T) {
	cat, ds := catalog.Load(os.DirFS("../.."))
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	files, err := Generate(cat)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(files[LibraryDoc])
	for _, want := range []string{
		"<!-- " + Marker + " -->",
		"## Native dependencies",
		"## crypto\n\n```go\nimport \"github.com/eugenioenko/goalchemy/lib/crypto\"\n```",
		"### AES256GCMEncrypt\n\n```go\nfunc AES256GCMEncrypt(key, nonce, data, aad []byte) ([]byte, error)\n```",
		"### Key.Close\n\n```go\nfunc (k *Key) Close()\n```",
		"### Canceled\n\n```go\nvar Canceled error\n```",
		"### Mutex.Lock\n\n```go\nfunc (m *Mutex) Lock()\n```",
		"func All(fns ...func())",
		"| `AES256GCMEncrypt` | yes | yes | yes | yes | yes | yes | yes |",
		"- **Gate**: `cooperative` (may suspend the calling task)",
		"- **Gate**: `sequential` (never suspends)",
		"- **Contract**: `lib.crypto.aes256_gcm_encrypt` 1.0.0",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("library reference lacks %q", want)
		}
	}
	if strings.Contains(doc, "native capability boundary") {
		t.Error("library reference contains placeholder summaries")
	}
	again, err := Generate(cat)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again[LibraryDoc], files[LibraryDoc]) {
		t.Error("library reference is not deterministic")
	}

	cat.FS = overlayFS{cat.FS, fstest.MapFS{LibraryDoc: {Data: []byte("stale\n")}}}
	found := false
	for _, p := range Check(cat, files) {
		if strings.HasPrefix(p, LibraryDoc+":") {
			found = true
		}
	}
	if !found {
		t.Error("Check does not flag a stale library reference")
	}
}
