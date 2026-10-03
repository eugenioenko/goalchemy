#!/bin/sh
# Compiles the Rust runtime and harness, then serves JSON Lines requests.
set -e
root=$(cd "$(dirname "$0")/../.." && pwd)
out=$(mktemp -d)
trap 'rm -rf "$out"' EXIT
mkdir -p "$out/rt"
cp -r "$root/types" "$root/runtime" "$out/rt/"
cp "$root"/tests/harness/*.rs "$out/"
{
  for f in "$out"/rt/types/*.rs "$out"/rt/runtime/*.rs; do
    d=$(basename "$(dirname "$f")"); n=$(basename "$f" .rs)
    printf '#[path = "%s/%s.rs"]\npub mod %s;\npub use %s::*;\n' "$d" "$n" "$n" "$n"
  done
} > "$out/rt/mod.rs"
cp "$root/tests/harness/Cargo.toml" "$out/Cargo.toml"
cp "$root/tests/harness/Cargo.lock" "$out/Cargo.lock"
CARGO_TARGET_DIR="$root/../../out/rust-tdf-library/sdk/target" cargo run --locked --offline --quiet --release --manifest-path "$out/Cargo.toml"
