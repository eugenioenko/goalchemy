#!/bin/sh
set -eu
repo=$(cd "$(dirname "$0")/../../.." && pwd)
gcdir=${GOALCHEMY_BDWGC:-$repo/.toolchains/bdwgc}
if [ -f "$gcdir/lib/libgc.a" ]; then
  set -- -I"$gcdir/include" "$gcdir/lib/libgc.a"
elif pkg-config --exists bdw-gc 2>/dev/null; then
  set -- $(pkg-config --cflags --libs bdw-gc)
else
  set -- -lgc
fi
out=${GOALCHEMY_BYTE_BENCH_OUT:-$repo/out/c-byte/benchmark}
mkdir -p "$out"
${CC:-cc} --version

printf 'build: CC=%s CFLAGS=%s bdwgc=%s\n' "${CC:-cc}" "${CFLAGS:--O2}" "$gcdir"
${CC:-cc} -std=gnu17 ${CFLAGS:--O2} -w -I"$repo/targets/c/types" \
  "$repo/targets/c/tests/byte_storage_bench.c" "$repo"/targets/c/types/*.c \
  "$repo"/targets/c/runtime/*.c "$@" -lpthread -o "$out/bench"
for n in 2097152 8388608; do
  for mode in native generic; do "$out/bench" "$mode" "$n"; done
done
