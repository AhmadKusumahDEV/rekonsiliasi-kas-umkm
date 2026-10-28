import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile, readdir } from 'node:fs/promises';
import { buildSite } from '../../tools/build-static.mjs';
import { join } from 'node:path';

test('public build keeps landing and app routes separate without publishing source or private files', async () => {
  const output = await buildSite();
  assert.deepEqual((await readdir(output)).sort(), ['app.html', 'assets', 'index.html', 'landing']);
  const home = await readFile(join(output, 'index.html'), 'utf8');
  const app = await readFile(join(output, 'app.html'), 'utf8');
  assert.match(home, /id="hero"/);
  assert.match(home, /href="\/app#register"/);
  assert.match(home, /href="\/app#login"/);
  assert.match(app, /assets\/app\.js/);
  assert.doesNotMatch(home, /assets\/app\.js/);
  assert.deepEqual(await readdir(join(output, 'landing')), ['assets']);
  const refs = [...home.matchAll(/(?:src|href)="(\/(?:assets|landing\/assets)\/[^"?]+)(?:\?[^" ]*)?"/g)];
  for (const [, path] of refs) assert.ok((await readFile(join(output, path.slice(1)))).length, path);
});
