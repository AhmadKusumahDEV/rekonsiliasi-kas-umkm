import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile, readdir } from 'node:fs/promises';
import { buildSite } from '../../tools/build-static.mjs';
import { join } from 'node:path';

test('public build keeps landing and app routes separate without publishing source or private files', async () => {
  const output = await buildSite();
  assert.deepEqual((await readdir(output)).sort(), ['app.html', 'assets', 'index.html', 'landing', 'pwa', 'sw.js']);
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
  const manifest = JSON.parse(await readFile(join(output, 'pwa', 'manifest.webmanifest'), 'utf8'));
  assert.equal(manifest.id, '/app');
  assert.equal(manifest.start_url, '/app');
  assert.equal(manifest.display, 'standalone');
  assert.deepEqual((await readdir(join(output, 'pwa'))).sort(), ['icons', 'manifest.webmanifest', 'offline.html', 'offline.js', 'pwa.css', 'register.js']);
  const worker = await readFile(join(output, 'sw.js'), 'utf8');
  assert.doesNotMatch(worker, /__BUILD_ID__/);
  assert.match(worker, /efa-public-[a-f0-9]{16}/);
  for (const [name,size] of [['icon-192',192],['icon-512',512],['maskable-512',512],['apple-touch-icon',180]]) {
    const png = await readFile(join(output, 'pwa', 'icons', name + '.png'));
    assert.equal(png.readUInt32BE(16), size);
    assert.equal(png.readUInt32BE(20), size);
  }
});
