#!/usr/bin/env bash
# Prepare the native runtime and real-browser prerequisites used by the CI suite.
set -euo pipefail
repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo"
case "${1:-}" in
  '') browser_deps=() ;;
  --with-browser-deps) browser_deps=(--with-deps) ;;
  *) echo 'usage: scripts/ci-bootstrap.sh [--with-browser-deps]' >&2; exit 2 ;;
esac

# Match tests that intentionally require repository-local, hash-verified tools.
bash scripts/fetch-toolchains.sh dotnet
bash scripts/fetch-toolchains.sh llvm

# Prewarm both the frontend's reference Go and the native HTTP probe's Go.
reference_go=$(sed -n 's/.*ReferenceToolchain = "\([^"]*\)".*/\1/p' internal/frontend/frontend.go)
[[ -n $reference_go ]] || { echo 'missing frontend reference toolchain' >&2; exit 1; }
GOTOOLCHAIN="$reference_go" go version
GOTOOLCHAIN=go1.25.14 go version

python3 -c 'import sys; assert sys.version_info[:2] == (3, 10), "CI Python lock requires CPython 3.10"'
python3 -m venv .toolchains/ci-python
.toolchains/ci-python/bin/python -m pip install --only-binary=:all: --require-hashes -r ci/python-requirements.txt

browser="$repo/out/typescript-host-operations/tooling"
mkdir -p "$browser"
cp ci/browser-tooling/package.json ci/browser-tooling/package-lock.json "$browser/"
npm ci --prefix "$browser"
npm ci --ignore-scripts --prefix "$repo/targets/typescript"
export PLAYWRIGHT_BROWSERS_PATH=${PLAYWRIGHT_BROWSERS_PATH:-"$repo/.toolchains/playwright"}
"$browser/node_modules/.bin/playwright" install "${browser_deps[@]}" chromium
# Existing browser fixtures otherwise prefer any system Chrome on PATH.
export GOALCHEMY_CHROME
GOALCHEMY_CHROME=$(node --input-type=module - "$browser" <<'JS'
import { pathToFileURL } from 'node:url';
const { chromium } = await import(pathToFileURL(process.argv[2] + '/node_modules/playwright/index.mjs'));
process.stdout.write(chromium.executablePath());
JS
)
[[ -x $GOALCHEMY_CHROME ]] || { echo 'pinned Chromium executable missing' >&2; exit 1; }

# The Rust harness is intentionally offline. Fetch its exact locked native graph first.
cargo fetch --locked --manifest-path targets/rust/tests/harness/Cargo.toml --target x86_64-unknown-linux-gnu
bash targets/java/tests/crypto-dependencies.sh

mkdir -p out/ci
# The environment is also usable by local checks; no machine profiles are changed.
{
  printf 'export PATH=%q:"$PATH"\n' "$repo/.toolchains/ci-python/bin:$browser/node_modules/.bin:$repo/.toolchains/dotnet"
  printf 'export DOTNET_ROOT=%q\n' "$repo/.toolchains/dotnet"
  printf 'export PLAYWRIGHT_BROWSERS_PATH=%q\n' "$PLAYWRIGHT_BROWSERS_PATH"
  printf 'export GOALCHEMY_CHROME=%q\n' "$GOALCHEMY_CHROME"
} > out/ci/environment.sh
if [[ -n ${GITHUB_PATH:-} && -n ${GITHUB_ENV:-} ]]; then
  printf '%s\n' "$repo/.toolchains/ci-python/bin" "$browser/node_modules/.bin" "$repo/.toolchains/dotnet" >> "$GITHUB_PATH"
  printf 'DOTNET_ROOT=%s\nPLAYWRIGHT_BROWSERS_PATH=%s\nGOALCHEMY_CHROME=%s\n' "$repo/.toolchains/dotnet" "$PLAYWRIGHT_BROWSERS_PATH" "$GOALCHEMY_CHROME" >> "$GITHUB_ENV"
fi
printf 'CI prerequisites ready; local shell: source %s/out/ci/environment.sh\n' "$repo"
