// optional maintenance tool; checked-in icons make normal builds dependency-free.
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { dirname, join, isAbsolute } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const target = join(root, 'pwa', 'icons');
const modulePath = process.env.PLAYWRIGHT_MODULE || 'playwright';
const { chromium } = await import(isAbsolute(modulePath) ? pathToFileURL(modulePath).href : modulePath);
const mark = await readFile(join(root, 'frontend', 'assets', 'efa-mark.svg'), 'utf8');
const paths = mark.slice(mark.indexOf('>') + 1, mark.lastIndexOf('</svg>')).replace(/<title[^>]*>.*?<\/title>/s, '');
function vector(scale, size = 512) {
  const offset = (512 - 512 * scale) / 2;
  return '<svg xmlns="http://www.w3.org/2000/svg" width="' + size + '" height="' + size + '" viewBox="0 0 512 512"><title>EFA — Easy Financial Audit</title><rect width="512" height="512" fill="#102b43"/><g transform="translate(' + offset + ' ' + offset + ') scale(' + scale + ')">' + paths + '</g></svg>';
}
await mkdir(target, { recursive: true });
await writeFile(join(target, 'icon.svg'), vector(.78));
await writeFile(join(target, 'maskable.svg'), vector(.66));
const browser = await chromium.launch({ channel: process.env.PWA_BROWSER_CHANNEL || 'chrome', headless: true });
try {
  const page = await browser.newPage({ viewport: { width: 512, height: 512 }, deviceScaleFactor: 1 });
  for (const [name, size, scale] of [['icon-192',192,.78],['icon-512',512,.78],['maskable-512',512,.66],['apple-touch-icon',180,.78]]) {
    await page.setContent('<style>body{margin:0}</style>' + vector(scale,size));
    await page.locator('svg').screenshot({ path: join(target, name + '.png') });
  }
} finally { await browser.close(); }
console.log('EFA PWA icons rendered.');
