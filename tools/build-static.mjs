// publish only public files; source folders stay separate and no API is bundled.
import { cp, mkdir, rm, lstat } from 'node:fs/promises';
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
  return output;
}
if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  console.log('Public site built:', await buildSite());
}
