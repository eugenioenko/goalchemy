#!/bin/sh
# Compiles the Java runtime and harness, then serves JSON Lines requests.
set -e
root=$(cd "$(dirname "$0")/../.." && pwd)
repo=$(cd "$root/../.." && pwd)
bin=""
for d in "$JAVA_HOME" "$repo"/.toolchains/jdk-*; do
  if [ -n "$d" ] && [ -x "$d/bin/javac" ]; then bin="$d/bin/"; fi
done
out=$(mktemp -d)
trap 'rm -rf "$out"' EXIT
"${bin}javac" -nowarn -encoding UTF-8 -d "$out" "$root"/types/*.java "$root"/runtime/*.java "$root"/tests/harness/*.java >&2
"${bin}java" -Xss512m -cp "$out" Harness
