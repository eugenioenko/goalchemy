#!/bin/sh
# Build against the actual collector and runtime; optional sanitizer flags are
# supplied by the default Go test for the independent structural suite.
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
. "$repo/targets/c/tests/native-deps.sh"
out=$(mktemp -d)
trap 'rm -rf "$out"' EXIT
${CC:-cc} -std=gnu17 ${CFLAGS:--O1} -w $GX_NATIVE_CFLAGS -DGX_BYTE_STANDALONE \
  -I"$repo/targets/c/types" \
  "$repo/targets/c/tests/byte_storage_test.c" "$repo"/targets/c/types/*.c \
  "$repo"/targets/c/runtime/*.c "$@" $GX_NATIVE_LIBS -lpthread -o "$out/test"
"$out/test"
for mode in 0 1 2 3 4 5 6 7 8; do "$out/test" gc "$mode"; done
for mode in fault-make fault-growth fault-generic-access; do
  set +e
  "$out/test" "$mode" > "$out/fault.log" 2>&1
  status=$?
  set -e
  # gx_fault aborts on this supported POSIX host. A segfault (139), sanitizer
  # rejection or ordinary failure cannot satisfy an implementation fault.
  if [ "$status" -ne 134 ]; then
    echo "expected SIGABRT implementation fault: $mode (status $status)" >&2
    cat "$out/fault.log" >&2; exit 1
  fi
  if ! rg -q '^goalchemy runtime fault: goalchemy fault: ((allocation|slice growth) exceeds host limits|native byte backing used as gx_V storage)$' "$out/fault.log"; then
    cat "$out/fault.log" >&2; exit 1
  fi
  if rg -q 'AddressSanitizer|UndefinedBehaviorSanitizer|runtime error:|Segmentation fault' "$out/fault.log"; then
    cat "$out/fault.log" >&2; exit 1
  fi
done
