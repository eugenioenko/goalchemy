#!/usr/bin/env bash
# Prepare the native runtime and real-browser prerequisites used by the CI suite.
set -euo pipefail
repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo"
# Read the same version pin used by the archive fetcher.
source "$repo/toolchains.lock"
browser_deps=()
suite=()
while (($#)); do
  case "$1" in
    --with-browser-deps) browser_deps=(--with-deps); shift ;;
    --suite) suite+=("${2:?missing suite}"); shift 2 ;;
    --target) [[ $# -ge 2 ]] || exit 2; suite+=(--target "$2"); shift 2 ;;
    *) echo 'usage: scripts/ci-bootstrap.sh [--suite <suite>] [--target <target>] [--with-browser-deps]' >&2; exit 2 ;;
  esac
done
features=$(python3 scripts/ci-suite.py "${suite[@]}" --setup | python3 -c 'import json,sys; print(" ".join(json.load(sys.stdin)["features"]))')
has() { [[ " $features " == *" $1 "* ]]; }
printf 'CI setup: %s\n' "$features"
paths=()
environment=()

# Match tests that intentionally require repository-local, hash-verified tools.
if has csharp; then
  bash scripts/fetch-toolchains.sh dotnet
  export DOTNET_ROOT="$repo/.toolchains/dotnet"
  paths+=("$DOTNET_ROOT")
  environment+=("DOTNET_ROOT=$DOTNET_ROOT")
fi
if has llvm; then bash scripts/fetch-toolchains.sh llvm; fi
if has swift; then
  bash scripts/fetch-toolchains.sh swift
  paths+=("$repo/.toolchains/swift-$SWIFT_VERSION/usr/bin")
fi

# Prewarm both the frontend's reference Go and the native HTTP probe's Go.
reference_go=$(sed -n 's/.*ReferenceToolchain = "\([^"]*\)".*/\1/p' internal/frontend/frontend.go)
[[ -n $reference_go ]] || { echo 'missing frontend reference toolchain' >&2; exit 1; }
GOTOOLCHAIN="$reference_go" go version
if has go-probe; then GOTOOLCHAIN=go1.25.14 go version; fi

if has python; then
  python3 -c 'import sys; assert sys.version_info[:2] == (3, 10), "CI Python lock requires CPython 3.10"'
  python3 -m venv .toolchains/ci-python
  .toolchains/ci-python/bin/python -m pip install --only-binary=:all: --require-hashes -r ci/python-requirements.txt
  paths+=("$repo/.toolchains/ci-python/bin")
fi

if has typescript; then
  npm ci --ignore-scripts --prefix "$repo/targets/typescript"
  paths+=("$repo/targets/typescript/node_modules/.bin")
fi
if has browser; then
  browser="$repo/out/typescript-host-operations/tooling"
  mkdir -p "$browser"
  cp ci/browser-tooling/package.json ci/browser-tooling/package-lock.json "$browser/"
  npm ci --prefix "$browser"
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
  paths+=("$browser/node_modules/.bin")
  environment+=("PLAYWRIGHT_BROWSERS_PATH=$PLAYWRIGHT_BROWSERS_PATH" "GOALCHEMY_CHROME=$GOALCHEMY_CHROME")
fi

# The Rust harness is intentionally offline. Fetch its exact locked native graph first.
if has rust; then
  cargo fetch --locked --manifest-path targets/rust/tests/harness/Cargo.toml --target x86_64-unknown-linux-gnu
fi
# Compile the full native graph before timed contracts and parallel tests begin.
if has rust-harness; then sh targets/rust/tests/harness/run.sh </dev/null; fi
if has java; then bash targets/java/tests/crypto-dependencies.sh; fi
if has csharp; then sh targets/csharp/tests/checksum-dependencies.sh; fi

mkdir -p out/ci
# The environment is also usable by local checks; no machine profiles are changed.
{
  for entry in "${paths[@]}"; do printf 'export PATH=%q:"$PATH"\n' "$entry"; done
  for entry in "${environment[@]}"; do printf 'export %q\n' "$entry"; done
} > out/ci/environment.sh
if [[ -n ${GITHUB_PATH:-} && -n ${GITHUB_ENV:-} ]]; then
  for entry in "${paths[@]}"; do printf '%s\n' "$entry"; done >> "$GITHUB_PATH"
  for entry in "${environment[@]}"; do printf '%s\n' "$entry"; done >> "$GITHUB_ENV"
fi
printf 'CI prerequisites ready; local shell: source %s/out/ci/environment.sh\n' "$repo"
