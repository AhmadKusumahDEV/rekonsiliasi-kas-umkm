// publish only public files; source folders stay separate and no API is bundled.
import { cp, mkdir, rm, lstat, readFile, writeFile, readdir } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { buildPanduan } from './build-panduan.mjs';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
export async function buildSite() {
  const output = join(root, 'dist');
  if (output !== resolve(root, 'dist')) throw new Error('Unexpected output directory');
  const existing = await lstat(output).catch((error) => {
    if (error.code !== 'ENOENT') throw error;
    return null;
  });
  if (existing?.isSymbolicLink()) throw new Error('Refusing to replace a linked output directory');
  await rm(output, { recursive: true, force: true });
  await mkdir(join(output, 'landing'), { recursive: true });
  // Exact .html verification URL is rewritten to a non-HTML asset to avoid cleanUrls redirects.
  await cp(join(root, 'google191985305be39b7b.html'), join(output, 'google191985305be39b7b.txt'));
  await cp(join(root, 'frontend', 'assets'), join(output, 'assets'), { recursive: true });
  await cp(join(root, 'frontend', 'index.html'), join(output, 'app.html'));
  await cp(join(root, 'landing', 'index.html'), join(output, 'index.html'));
  await cp(join(root, 'landing', 'assets'), join(output, 'landing', 'assets'), { recursive: true });
  await mkdir(join(output, 'artikel'), { recursive: true });
  for (const file of await readdir(join(root, 'landing', 'articles'))) {
    if (/^[a-z0-9-]+\.html$/.test(file)) await cp(join(root, 'landing', 'articles', file), join(output, 'artikel', file));
  }
  await mkdir(join(output, 'pwa'), { recursive: true });
  for (const file of ['manifest.webmanifest', 'register.js', 'pwa.css', 'offline.html', 'offline.js']) {
    await cp(join(root, 'pwa', file), join(output, 'pwa', file));
  }
  await cp(join(root, 'pwa', 'icons'), join(output, 'pwa', 'icons'), { recursive: true });
  // Analytics is public-page only; authenticated app content is never instrumented.
  for (const file of ['index.html', ...((await readdir(join(output, 'artikel'))).map(name => 'artikel/' + name))]) {
    const path = join(output, file);
    const html = await readFile(path, 'utf8');
    // Windows copies can retain Hidden; replace generated HTML instead of truncating it.
    await rm(path, { force: true });
    await writeFile(path, html.replace('</head>', '<script defer src="/landing/assets/analytics.js"></script></head>'));
  }
  await buildPanduan(root, output);
  await writeFile(join(output, 'robots.txt'), 'User-agent: *\nAllow: /\nDisallow: /app\nSitemap: https://efa.madgo.my.id/sitemap.xml\n');
  const guides = JSON.parse(await readFile(join(output,'panduan/search-index.json'),'utf8')).map(x=>x.url.slice(1));
  const pages = ['', ...((await readdir(join(output, 'artikel'))).map(name => 'artikel/' + name.replace(/\.html$/, ''))), ...guides];
  await writeFile(join(output, 'sitemap.xml'), '<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">' + pages.map(path => '<url><loc>https://efa.madgo.my.id/' + path + '</loc></url>').join('') + '</urlset>');
  // changing any published asset updates the worker, without a manual cache bump.
  const hash = createHash('sha256');
  async function fingerprint(directory) {
    for (const file of (await readdir(directory, { withFileTypes: true })).sort((a,b) => a.name.localeCompare(b.name))) {
      const path = join(directory, file.name);
      if (file.isDirectory()) await fingerprint(path);
      else { hash.update(path.slice(output.length).replaceAll('\\', '/')); hash.update(await readFile(path)); }
    }
  }
  await fingerprint(output);
  const worker = await readFile(join(root, 'pwa', 'service-worker.js'), 'utf8');
  hash.update(worker);
  await writeFile(join(output, 'sw.js'), worker.replaceAll('__BUILD_ID__', hash.digest('hex').slice(0,16)));
  return output;
}
if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  console.log('Public site built:', await buildSite());
}
