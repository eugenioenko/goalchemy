// Actual Chromium execution, with a bundle graph that rejects Node globals/imports.
// Pinned tooling is installed under ignored out/typescript-host-operations/tooling.
import { createServer } from 'node:http';
import { execFileSync } from 'node:child_process';
import { readFile } from 'node:fs/promises';
import { build } from '../../../out/typescript-host-operations/tooling/node_modules/esbuild/lib/main.js';
import { chromium } from '../../../out/typescript-host-operations/tooling/node_modules/playwright/index.mjs';
const result = await build({entryPoints:['targets/typescript/tests/host_operations_suite.ts'],bundle:true,platform:'browser',format:'esm',write:false,metafile:true});
for(const file of Object.keys(result.metafile.inputs)) {
  const source=await readFile(file,'utf8');
  if(/node:|\bBuffer\b|\bprocess\b/.test(source)) throw new Error('Node dependency in portable graph: '+file);
}
const bundle=result.outputFiles[0].text;
let emittedBundle=null;
if(process.argv[2]) {
  const emitted=await build({entryPoints:[process.argv[2]+'/browser-test-only.ts'],bundle:true,platform:'browser',format:'esm',write:false,metafile:true});
  for(const file of Object.keys(emitted.metafile.inputs)) {
    const source=await readFile(file,'utf8');
    if(/node:|\bBuffer\b|\bprocess\b/.test(source))throw new Error('Node dependency in emitted graph: '+file);
  }
  emittedBundle=emitted.outputFiles[0].text;
  console.log('Emitted portable graph files',Object.keys(emitted.metafile.inputs).length);
}
const server=createServer((req,res)=>{
  if(req.url==='/host') { const chunks=[]; req.on('data',b=>chunks.push(b));req.on('end',()=>setTimeout(()=>res.end(Buffer.concat(chunks)),15));return; }
  if(req.url==='/emitted.js') {res.setHeader('Content-Type','text/javascript');res.end(emittedBundle);return;}
  res.setHeader('Content-Type',req.url==='/suite.js'?'text/javascript':'text/html');
  res.end(req.url==='/suite.js'?bundle:'<script type="module">import {runPortableSuite,runTestOnlyFetch} from "/suite.js"; window.result=(async()=>{const results=await runPortableSuite();await runTestOnlyFetch("/host");return [...results,"genuine browser fetch and unrelated progress (test only transport)"];})();</script>');
});
await new Promise(r=>server.listen(0,'127.0.0.1',r));
let browser;
try {
  let executablePath=process.env.GOALCHEMY_CHROME;
  if(!executablePath) for(const name of ['google-chrome','chromium','chromium-browser']) {
    try {executablePath=execFileSync('which',[name],{encoding:'utf8',stdio:['ignore','pipe','ignore']}).trim();break;}catch{}
  }
  browser=await chromium.launch({...(executablePath?{executablePath}:{}),headless:true,args:['--no-sandbox']});
  const page=await browser.newPage();const failures=[];page.on('pageerror',e=>failures.push(String(e)));
  console.log('Browser',await browser.version());console.log('Playwright 1.58.2; esbuild 0.25.12; portable graph files',Object.keys(result.metafile.inputs).length);
  await page.goto('http://127.0.0.1:'+server.address().port);
  const passed=await page.evaluate(()=>window.result);
  if(emittedBundle) {
    await page.evaluate(async()=>{const {runEmitted}=await import('/emitted.js');await runEmitted();});
    passed.push('actual emitted cooperative Promise frames/source context/HTTP/import');
  }
  if(failures.length) throw new Error(failures.join('\n'));
  for(const name of passed) console.log('PASS',name);
  console.log('PASS graph has no Node imports/Buffer/process and no shims');
} finally {await browser?.close();await new Promise(r=>server.close(r));}
