#!/bin/sh
# Compiles the C runtime and harness, then serves JSON Lines requests.
set -e
root=$(cd "$(dirname "$0")/../.." && pwd)
repo=$(cd "$root/../.." && pwd)
gcdir=${GOALCHEMY_BDWGC:-$repo/.toolchains/bdwgc}
if [ -f "$gcdir/lib/libgc.a" ]; then
  gc="-I$gcdir/include $gcdir/lib/libgc.a"
elif pkg-config --exists bdw-gc 2>/dev/null; then
  gc=$(pkg-config --cflags --libs bdw-gc)
else
  gc=-lgc
fi
. "$repo/targets/c/tests/native-deps.sh"
out=$(mktemp -d)
trap 'rm -rf "$out"' EXIT
${CC:-cc} -std=gnu17 -O1 -w $GX_NATIVE_CFLAGS -I"$root/types" -I"$root/tests/harness" -o "$out/harness" \
  "$root"/types/*.c "$root"/runtime/*.c "$root"/tests/harness/*.c $gc $GX_NATIVE_LIBS -lpthread >&2
"$out/harness"
