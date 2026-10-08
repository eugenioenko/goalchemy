#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
set -eu
root=$(cd "$(dirname "$0")/../.." && pwd)
repo=$(cd "$root/../.." && pwd)
swift_compiler=${SWIFTC:-swiftc}
if ! command -v "$swift_compiler" >/dev/null 2>&1; then
    for candidate in "$repo"/.toolchains/swift-*/usr/bin/swiftc; do
        if [ -x "$candidate" ]; then swift_compiler=$candidate; fi
    done
fi
if [ -d "$repo/.toolchains/pkgconfig" ]; then
    export PKG_CONFIG_PATH="$repo/.toolchains/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
fi
out=$(mktemp -d)
trap 'rm -rf "$out"' EXIT
${CC:-cc} -Wno-deprecated-declarations -O2 -c "$root/native/GoalchemyNative.c" -o "$out/native.o" $(pkg-config --cflags openssl zlib libcurl) >&2
"$swift_compiler" -swift-version 5 -suppress-warnings -I "$root/native" "$root/tests/harness/main.swift" "$root/tests/harness/Codec.swift" "$root/tests/harness/HarnessGen.swift" "$root"/types/*.swift "$root"/runtime/*.swift "$out/native.o" $(pkg-config --libs openssl zlib libcurl) -o "$out/harness" >&2
"$out/harness"
