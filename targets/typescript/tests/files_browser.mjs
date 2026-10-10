// Actual browser execution of a generated library that uses std/os: browsers
// have no file system, so every call must fail with an unsupported error.
import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';

const tooling = resolve(process.env.TDF_BROWSER_TOOLING || 'out/typescript-host-operations/tooling/node_modules');
const { build } = await import(pathToFileURL(tooling + '/esbuild/lib/main.js'));
const { chromium } = await import(pathToFileURL(tooling + '/playwright/index.mjs'));
const generated = process.argv[2];
const suite = `import * as g from 'goalchemy-generated';
export async function run(){
  const out=[];
  out.push(await g.Read('key.pem'));
  out.push(await g.Write('out.bin','x'));
  return out;
}`;
const result = await build({
  stdin: { resolveDir: generated, contents: suite },
  bundle: true, platform: 'browser', format: 'esm', write: false, metafile: true,
});
for (const file of Object.keys(result.metafile.inputs)) {
  const source = file === '<stdin>' ? suite : await readFile(resolve(generated, file), 'utf8').catch(() => readFile(file, 'utf8'));
  if (/node:|\bBuffer\b|\bprocess\b/.test(source)) throw new Error('Node dependency in browser graph: ' + file);
}
const server = createServer((req, res) => {
  res.setHeader('Content-Type', req.url === '/suite.js' ? 'text/javascript' : 'text/html');
  res.end(req.url === '/suite.js' ? result.outputFiles[0].text :
    '<script type="module">import*as suite from"/suite.js";window.result=suite.run();</script>');
});
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
let browser;
try {
  browser = await chromium.launch({
    ...(process.env.GOALCHEMY_CHROME ? { executablePath: process.env.GOALCHEMY_CHROME } : {}),
    headless: true, args: ['--no-sandbox'],
  });
  const page = await browser.newPage();
  const failures = [];
  page.on('pageerror', error => failures.push(String(error)));
  await page.goto('http://127.0.0.1:' + server.address().port);
  const got = await page.evaluate(() => window.result);
  const want = ['error: open key.pem: operation not supported', 'error: open out.bin: operation not supported'];
  if (failures.length || JSON.stringify(got) !== JSON.stringify(want))
    throw new Error('browser file failures: ' + failures + '; got ' + JSON.stringify(got));
  console.log('PASS Chromium ' + await browser.version() + ': std/os reports unsupported without a file system');
} finally {
  await browser?.close();
  await new Promise(resolve => server.close(resolve));
}
