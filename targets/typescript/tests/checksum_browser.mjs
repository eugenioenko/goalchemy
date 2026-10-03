// Actual browser execution; Node tooling is outside the production bundle.
import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';

const tooling = resolve(process.env.TDF_BROWSER_TOOLING || 'out/typescript-host-operations/tooling/node_modules');
const { build } = await import(pathToFileURL(tooling + '/esbuild/lib/main.js'));
const { chromium } = await import(pathToFileURL(tooling + '/playwright/index.mjs'));
const result = await build({
  entryPoints: ['targets/typescript/tests/checksum_suite.ts'],
  bundle: true, platform: 'browser', format: 'esm', write: false, metafile: true,
});
for (const file of Object.keys(result.metafile.inputs)) {
  const source = await readFile(file, 'utf8');
  if (/node:|\bBuffer\b|\bprocess\b/.test(source)) throw new Error('Node dependency in browser graph: ' + file);
}
const server = createServer((req,res) => {
  res.setHeader('Content-Type', req.url === '/suite.js' ? 'text/javascript' : 'text/html');
  res.end(req.url === '/suite.js' ? result.outputFiles[0].text :
    '<script type="module">import{runCRC32Suite}from"/suite.js";window.result=runCRC32Suite();</script>');
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
  const count = await page.evaluate(() => window.result);
  if (failures.length || count !== 60) throw new Error('CRC browser failures: ' + failures + '; cases=' + count);
  console.log('PASS Chromium ' + await browser.version() + ': ' + count + ' CRC cases; portable graph files=' + Object.keys(result.metafile.inputs).length);
} finally {
  await browser?.close();
  await new Promise(resolve => server.close(resolve));
}
