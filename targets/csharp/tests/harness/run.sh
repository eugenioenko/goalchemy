#!/bin/sh
# Compiles the C# runtime and harness, then serves JSON Lines requests.
set -e
root=$(cd "$(dirname "$0")/../.." && pwd)
repo=$(cd "$root/../.." && pwd)
dotnet=dotnet
if [ -n "$DOTNET_ROOT" ] && [ -x "$DOTNET_ROOT/dotnet" ]; then dotnet="$DOTNET_ROOT/dotnet"
elif [ -x "$repo/.toolchains/dotnet/dotnet" ]; then dotnet="$repo/.toolchains/dotnet/dotnet"; fi
droot=$(dirname "$(readlink -f "$(command -v "$dotnet")")")
sdk=$("$dotnet" --list-sdks | awk '/^8\./ {v=$1} END {print v}')
ref=$(ls -d "$droot"/packs/Microsoft.NETCore.App.Ref/8.*/ref/net8.0 | tail -n 1)
out=$(mktemp -d)
trap 'rm -rf "$out"' EXIT
refs=""
for f in "$ref"/*.dll; do refs="$refs -r:$f"; done
"$dotnet" "$droot/sdk/$sdk/Roslyn/bincore/csc.dll" -nologo -noconfig -nostdlib -nowarn:CS0162,CS0164,CS0168,CS0219,CS8981 \
  -langversion:12 -nullable:disable -out:"$out/harness.dll" $refs \
  "$root"/types/*.cs "$root"/runtime/*.cs "$root"/tests/harness/*.cs >&2
cat > "$out/harness.runtimeconfig.json" <<'JSON'
{"runtimeOptions": {"tfm": "net8.0", "framework": {"name": "Microsoft.NETCore.App", "version": "8.0.0"}}}
JSON
DOTNET_ROOT="$droot" "$dotnet" "$out/harness.dll"
