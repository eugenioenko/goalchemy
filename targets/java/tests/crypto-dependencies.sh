#!/usr/bin/env bash
set -euo pipefail
ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
DEST=${1:-"$ROOT/out/java-dependencies"}
mkdir -p "$DEST"
python3 - "$ROOT/targets/java/dependencies.lock.json" "$DEST" <<'PY'
import json,sys,pathlib,hashlib,urllib.request,tempfile,os
for d in json.loads(pathlib.Path(sys.argv[1]).read_text())['dependencies']:
 p=pathlib.Path(sys.argv[2])/f"{d['artifact']}-{d['version']}.jar"
 if not p.exists():
  fd,name=tempfile.mkstemp(prefix=p.name+'.',suffix='.part',dir=p.parent);os.close(fd);part=pathlib.Path(name)
  try:
   urllib.request.urlretrieve(d['url'],part)
   if hashlib.sha256(part.read_bytes()).hexdigest()!=d['sha256']:raise SystemExit('Java downloaded dependency checksum mismatch')
   part.replace(p)
  finally:part.unlink(missing_ok=True)
 if hashlib.sha256(p.read_bytes()).hexdigest()!=d['sha256']:raise SystemExit('Java dependency checksum mismatch: '+str(p))
 print(p.resolve())
PY
