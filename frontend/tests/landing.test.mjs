import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile, readdir } from 'node:fs/promises';
import { buildSite } from '../../tools/build-static.mjs';
import { join } from 'node:path';

test('public build keeps landing and app routes separate without publishing source or private files', async () => {
  const output = await buildSite();
  assert.deepEqual((await readdir(output)).sort(), ['app.html', 'artikel', 'assets', 'google191985305be39b7b.txt', 'index.html', 'landing', 'panduan', 'pwa', 'robots.txt', 'sitemap.xml', 'sw.js']);
  const home = await readFile(join(output, 'index.html'), 'utf8');
  const articles = await readdir(join(output, 'artikel'));
  assert.equal(articles.length, 3);
  for (const file of articles) {
    const article = await readFile(join(output, 'artikel', file), 'utf8');
    assert.match(article, /class="article-prose"/);
    assert.match(article, /instagram.com\/easyfinancialaudit/);
    assert.doesNotMatch(article, /id="pause-slider"|landing\.js/);
  }
  const funding = await readFile(join(output, 'artikel', 'rekonsiliasi-penjualan-umkm-2.html'), 'utf8');
  assert.match(funding, /Sulit Dapat Pinjaman Modal Usaha/);
  assert.match(funding, /canonical" href="https:\/\/efa.madgo.my.id\/artikel\/rekonsiliasi-penjualan-umkm-2"/);
  assert.doesNotMatch(funding, /87%|alasan nomor satu|tanpa takut salah cocok/);
  const app = await readFile(join(output, 'app.html'), 'utf8');
  assert.equal((await readFile(join(output, 'google191985305be39b7b.txt'), 'utf8')).trim(), 'google-site-verification: google191985305be39b7b.html');
  assert.match(home, /analytics\.js/);
  assert.doesNotMatch(app, /analytics\.js|googletagmanager/);
  assert.match(await readFile(join(output, 'sitemap.xml'), 'utf8'), /saldo-rekening-bukan-laba/);
  assert.match(await readFile(join(output, 'robots.txt'), 'utf8'), /Disallow: \/app/);
  assert.match(home, /id="hero"/);
  assert.match(home, /href="\/panduan\/">Panduan lengkap/);
  const guideIndex = JSON.parse(await readFile(join(output,'panduan/search-index.json'),'utf8'));
  assert.equal(guideIndex.length,16);
  for (const page of guideIndex) {
    const html = await readFile(join(output,page.url,'index.html'),'utf8');
    assert.doesNotMatch(html, /<script\b(?![^>]*\bsrc=)[^>]*>\s*[^<\s]/i, page.url+' inline script');
    assert.doesNotMatch(html, /\sstyle="|<style\b/i, page.url+' inline style');
    assert.match(html, /efa-guide-search/);
  }
  assert.match(await readFile(join(output,'panduan/version.json'),'utf8'), /"version":"[a-f0-9]{16}"/);
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
