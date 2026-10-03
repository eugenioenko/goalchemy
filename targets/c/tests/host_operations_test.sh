#!/bin/sh
set -eu
repo=$(cd "$(dirname "$0")/../../.." && pwd)
gcdir=${GOALCHEMY_BDWGC:-$repo/.toolchains/bdwgc}
if [ -f "$gcdir/lib/libgc.a" ]; then set -- -I"$gcdir/include" "$gcdir/lib/libgc.a"
elif pkg-config --exists bdw-gc 2>/dev/null; then set -- $(pkg-config --cflags --libs bdw-gc)
else set -- -lgc; fi
. "$repo/targets/c/tests/native-deps.sh"
out=$(mktemp -d)
trap 'rm -rf "$out"' EXIT
# Constructor-failure injection is confined to this copied runtime source.
python3 - "$repo/targets/c/types/sched.c" "$out/sched.c" <<'PYTEST'
import sys
s=open(sys.argv[1]).read();marker='    gx_Frame *f = GC_MALLOC(sizeof(gx_Frame));'
assert s.count(marker)==1
s=s.replace(marker,'    extern bool test_only_fail_frame(void);\n    if (test_only_fail_frame()) gx_throw(gx_cstr("test-only harness frame constructor failure"));\n'+marker)
open(sys.argv[2],'w').write(s)
PYTEST
types=""
for source in "$repo"/targets/c/types/*.c; do
 if [ "$(basename "$source")" != sched.c ]; then types="$types $source"; fi
done
${CC:-cc} -std=gnu17 ${CFLAGS:--O1 -g} -w $GX_NATIVE_CFLAGS -I"$repo/targets/c/types" \
 "$repo/targets/c/tests/host_operations_test.c" "$out/sched.c" $types \
 "$repo"/targets/c/runtime/*.c "$@" $GX_NATIVE_LIBS -lpthread -o "$out/test"
"$out/test"
