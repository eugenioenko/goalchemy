#!/bin/sh
# Restore only the official, content-hash-locked Microsoft CRC dependency.
set -eu
target=$(cd "$(dirname "$0")/.." && pwd)
repo=$(cd "$target/../.." && pwd)
dotnet=dotnet
if [ -n "${DOTNET_ROOT:-}" ] && [ -x "$DOTNET_ROOT/dotnet" ]; then dotnet="$DOTNET_ROOT/dotnet"
elif [ -x "$repo/.toolchains/dotnet/dotnet" ]; then dotnet="$repo/.toolchains/dotnet/dotnet"; fi
"$dotnet" restore "$target/checksum.csproj" --locked-mode --nologo \
  -p:BaseIntermediateOutputPath="$repo/out/csharp-checksum-dependencies/obj/" >&2
packages=$(DOTNET_CLI_UI_LANGUAGE=en "$dotnet" nuget locals global-packages --list | sed -n 's/^global-packages: //p')
dll="$packages/system.io.hashing/8.0.0/lib/net8.0/System.IO.Hashing.dll"
test -f "$dll"
printf '%s\n' "$dll"
