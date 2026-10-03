package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/testutil"
)

func TestGeneratedRustChecksumLibrary(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source, out, consumer := t.TempDir(), t.TempDir(), t.TempDir()
	write := func(dir, name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(source, "go.mod", fmt.Sprintf("module crcprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root))
	write(source, "library.go", `package crcprobe
import("github.com/eugenioenko/goalchemy/lib/checksum";"github.com/eugenioenko/goalchemy/lib/context")
func CRC32IEEE(ctx context.Context,data []byte)(uint32,error){return checksum.CRC32IEEE(data),nil}
func SliceCRC32IEEE(ctx context.Context,data []byte)(uint32,error){return checksum.CRC32IEEE(data[1:len(data)-1]),nil}
`)
	if ds := testutil.CompileGate(source, "rust", out, "cooperative"); len(ds) > 0 {
		t.Fatal(ds)
	}
	write(consumer, "Cargo.toml", fmt.Sprintf(`[package]
name = "crc-consumer"
version = "0.1.0"
edition = "2021"
[dependencies]
goalchemy-generated = { path = %q }
[[bin]]
name = "crc-consumer"
path = "main.rs"
`, out))
	write(consumer, "main.rs", `use goalchemy_generated::{CRC32IEEE,SliceCRC32IEEE,CallOptions};
// Independent bitwise IEEE oracle; no CRC dependency in the consumer.
fn oracle(bytes:&[u8])->u32 {
 let mut crc=0xffffffffu32;
 for byte in bytes {crc^=*byte as u32;for _ in 0..8 {crc=(crc>>1)^if crc&1!=0 {0xedb88320}else{0};}}
 !crc
}
fn main(){
 assert_eq!(CRC32IEEE(Vec::new(),CallOptions::default()).wait().unwrap(),0);
 assert_eq!(CRC32IEEE(b"123456789".to_vec(),CallOptions::default()).wait().unwrap(),0xcbf43926u32);
 let mut state=0x8badf00du32;
 for n in [0,1,7,8,9,31,32,4097,65537,1048576] {
  let data:Vec<u8>=(0..n).map(|_|{state=state.wrapping_mul(1664525).wrapping_add(1013904223);(state>>24)as u8}).collect();
  let expected=oracle(&data);
  assert_eq!(CRC32IEEE(data.clone(),CallOptions::default()).wait().unwrap(),expected);
  let mut envelope=Vec::with_capacity(n+32);envelope.push(120);envelope.extend_from_slice(&data);envelope.push(121);
  assert_eq!(SliceCRC32IEEE(envelope,CallOptions::default()).wait().unwrap(),expected);
 }
 println!("PASS generated Rust CRC library consumer: independent IEEE oracle, offsets, unsigned results, boundaries through 1 MiB");
}
`)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "cargo", "run", "--release", "--offline", "--quiet")
	cmd.Dir = consumer
	cmd.Env = append(driver.ToolEnv(), "CARGO_TARGET_DIR="+filepath.Join(root, "out/rust-tdf-library/sdk/target"))
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated Rust Cargo library consumer: %v\n%s", err, data)
	}
	if !strings.Contains(string(data), "PASS generated Rust CRC library consumer") {
		t.Fatal(string(data))
	}
	t.Log(string(data))
}
