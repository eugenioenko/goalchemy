#!/bin/sh
# Compile once, then use a fresh .NET process for each diagnostic measurement.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)
cd "$repo"
droot=${DOTNET_ROOT:-"$repo/.toolchains/dotnet"}
dotnet="$droot/dotnet"
sdk=$("$dotnet" --list-sdks | awk '/^8\./ {v=$1} END {print v}')
ref=$(ls -d "$droot"/packs/Microsoft.NETCore.App.Ref/8.*/ref/net8.0 | tail -n 1)
out=out/csharp-byte-bench
mkdir -p "$out"
refs=""
for f in "$ref"/*.dll; do refs="$refs -r:$f"; done
hashing=$(sh targets/csharp/tests/checksum-dependencies.sh)
refs="$refs -r:$hashing"
cp "$hashing" "$out/System.IO.Hashing.dll"
"$dotnet" "$droot/sdk/$sdk/Roslyn/bincore/csc.dll" -nologo -noconfig -nostdlib -langversion:12 -nullable:disable -optimize+ \
    -out:"$out/bench.dll" $refs targets/csharp/types/*.cs targets/csharp/runtime/*.cs \
    targets/csharp/tests/ByteStorageBench.cs
cat > "$out/bench.runtimeconfig.json" <<'JSON'
{"runtimeOptions": {"tfm": "net8.0", "framework": {"name": "Microsoft.NETCore.App", "version": "8.0.0"}}}
JSON
"$dotnet" "$out/bench.dll" native 2
"$dotnet" "$out/bench.dll" boxed 2
"$dotnet" "$out/bench.dll" native 8
"$dotnet" "$out/bench.dll" boxed 8
