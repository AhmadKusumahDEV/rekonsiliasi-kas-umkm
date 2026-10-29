// publish only public files; source folders stay separate and no API is bundled.
import { cp, mkdir, rm, lstat, readFile, writeFile, readdir } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

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
  await cp(join(root, 'frontend', 'assets'), join(output, 'assets'), { recursive: true });
  await cp(join(root, 'frontend', 'index.html'), join(output, 'app.html'));
  await cp(join(root, 'landing', 'index.html'), join(output, 'index.html'));
  await cp(join(root, 'landing', 'assets'), join(output, 'landing', 'assets'), { recursive: true });
  await mkdir(join(output, 'pwa'), { recursive: true });
  for (const file of ['manifest.webmanifest', 'register.js', 'pwa.css', 'offline.html', 'offline.js']) {
    await cp(join(root, 'pwa', file), join(output, 'pwa', file));
  }
  await cp(join(root, 'pwa', 'icons'), join(output, 'pwa', 'icons'), { recursive: true });
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
