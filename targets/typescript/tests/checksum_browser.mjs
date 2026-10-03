// Actual browser execution; Node tooling is outside the production bundle.
import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';

const tooling = resolve(process.env.TDF_BROWSER_TOOLING || 'out/typescript-host-operations/tooling/node_modules');
const { build } = await import(pathToFileURL(tooling + '/esbuild/lib/main.js'));
const { chromium } = await import(pathToFileURL(tooling + '/playwright/index.mjs'));
const generated = process.argv[2];
const generatedSuite = `export {runCRC32Suite} from ${JSON.stringify(resolve('targets/typescript/tests/checksum_suite.ts'))};
import * as g from 'goalchemy-generated';
export async function runGeneratedCRC32Suite(){
  if(await g.CRC32IEEE(null)!==0)throw new Error('generated nil CRC');
  if(await g.CRC32IEEE(new Uint8Array())!==0)throw new Error('generated empty CRC');
  const backing=new Uint8Array([0,120,49,50,51,52,53,54,55,56,57,121,255]);
  const before=backing.slice();
  if(await g.SliceCRC32IEEE(backing.subarray(1,12))!==0xcbf43926)throw new Error('generated slice/unsigned CRC');
  if(backing.some((b,i)=>b!==before[i]))throw new Error('generated input mutated');
  return 3;
}`;
const result = await build({
  ...(generated ? { stdin: {
    resolveDir: generated,
    contents: generatedSuite,
  }} : { entryPoints: ['targets/typescript/tests/checksum_suite.ts'] }),
  bundle: true, platform: 'browser', format: 'esm', write: false, metafile: true,
});
for (const file of Object.keys(result.metafile.inputs)) {
  const source = file === '<stdin>' ? generatedSuite : await readFile(file, 'utf8');
  if (/node:|\bBuffer\b|\bprocess\b/.test(source)) throw new Error('Node dependency in browser graph: ' + file);
}
const server = createServer((req,res) => {
  res.setHeader('Content-Type', req.url === '/suite.js' ? 'text/javascript' : 'text/html');
  res.end(req.url === '/suite.js' ? result.outputFiles[0].text :
    '<script type="module">import*as suite from"/suite.js";window.result=Promise.resolve().then(async()=>({portable:suite.runCRC32Suite(),generated:suite.runGeneratedCRC32Suite?await suite.runGeneratedCRC32Suite():0}));</script>');
});
await new Promise(resolve => server.listen(0,'127.0.0.1',resolve));
let browser;
try {
  browser = await chromium.launch({
    ...(process.env.GOALCHEMY_CHROME ? { executablePath: process.env.GOALCHEMY_CHROME } : {}),
    headless: true, args: ['--no-sandbox'],
  });
  const page = await browser.newPage();
  const failures = [];
  page.on('pageerror',error => failures.push(String(error)));
  await page.goto('http://127.0.0.1:' + server.address().port);
  const counts = await page.evaluate(() => window.result);
  if (failures.length || counts.portable !== 60 || counts.generated !== (generated ? 3 : 0))
    throw new Error('CRC browser failures: ' + failures + '; cases=' + JSON.stringify(counts));
  console.log('PASS Chromium ' + await browser.version() + ': ' + counts.portable + ' portable/' + counts.generated + ' generated CRC cases; portable graph files=' + Object.keys(result.metafile.inputs).length);
} finally {
  await browser?.close();
  await new Promise(resolve => server.close(resolve));
}
