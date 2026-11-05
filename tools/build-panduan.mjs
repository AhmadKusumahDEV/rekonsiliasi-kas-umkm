import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { readFile, writeFile, readdir, mkdir, cp } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { join } from 'node:path';

// Keep the app dependency-free; only this separate documentation build uses Astro.
export async function buildPanduan(root, output) {
  const site = join(root, 'docs-site');
  await promisify(execFile)(process.execPath, [join(site, 'node_modules/astro/astro.js'), 'build'], {
    cwd: site, maxBuffer: 8 * 1024 * 1024, env: { ...process.env, ASTRO_TELEMETRY_DISABLED: '1' },
  });
  const built = join(site, 'dist');
  const documents = [];
  async function indexContent(directory) {
    for (const entry of await readdir(directory, { withFileTypes: true })) {
      const path = join(directory, entry.name);
      if(entry.isDirectory()) { await indexContent(path); continue; }
      if(!/\.mdx?$/.test(path)) continue;
      const text = await readFile(path, 'utf8');
      const title = JSON.parse(text.match(/^title: (.+)$/m)[1]);
      const description = JSON.parse(text.match(/^description: (.+)$/m)[1]);
      const slug = path.slice(join(site,'src/content/docs').length+1).replaceAll('\\','/').replace(/\.mdx?$/, '').replace(/^index$/, '');
      documents.push({title,description,url:'/panduan/'+(slug?slug+'/':''),text:text.replace(/^---[\s\S]*?---/, '').replace(/<[^>]*>/g,' ').replace(/import .+;/g,' ').replace(/\s+/g,' ')});
    }
  }
  await indexContent(join(site,'src/content/docs'));
  await writeFile(join(built,'search-index.json'), JSON.stringify(documents));
  const scripts = join(built, '_csp');
  await mkdir(scripts, { recursive: true });
  const extraStyles = new Map();
  const digest = content => createHash('sha256').update(content).digest('hex').slice(0, 20);
  async function visit(directory) {
    for (const entry of await readdir(directory, { withFileTypes: true })) {
      const path = join(directory, entry.name);
      if (entry.isDirectory()) { await visit(path); continue; }
      if (!entry.name.endsWith('.html')) continue;
      let html = await readFile(path, 'utf8');
      // Theme and sidebar bootstraps are executable inline scripts in Starlight.
      // Publish unchanged bytes as same-origin files: no unsafe-inline or nonces.
      const blocks = [...html.matchAll(/<script\b([^>]*)>([\s\S]*?)<\/script>/gi)];
      for (const block of blocks) {
        const [, attrs, content] = block;
        if (/\bsrc\s*=/.test(attrs) || !content.trim() || /type=["']application\/(?:ld\+)?json/.test(attrs)) continue;
        const name = digest(content) + '.js';
        await writeFile(join(scripts, name), content);
        html = html.replace(block[0], `<script${attrs} src="/panduan/_csp/${name}"></script>`);
      }
      // Static style attributes are also moved out, retaining strict style-src.
      html = html.replace(/<([a-z][a-z0-9-]*)(\s[^>]*?)>/gi, (tag, name, attrs) => {
        const match = attrs.match(/\sstyle="([^"]*)"/);
        if (!match) return tag;
        const className = 'csp-' + digest(match[1]);
        extraStyles.set(className, match[1]);
        attrs = attrs.replace(match[0], '');
        attrs = /\bclass="/.test(attrs) ? attrs.replace(/\bclass="/, `class="${className} `) : attrs + ` class="${className}"`;
        return `<${name}${attrs}>`;
      });
      const styles = [...html.matchAll(/<style\b[^>]*>([\s\S]*?)<\/style>/gi)];
      for (const block of styles) {
        const name = digest(block[1]) + '.css';
        await writeFile(join(scripts, name), block[1]);
        html = html.replace(block[0], `<link rel="stylesheet" href="/panduan/_csp/${name}">`);
      }
      html = html.replace('</head>', '<link rel="stylesheet" href="/panduan/_csp/static.css"></head>');
      await writeFile(path, html);
    }
  }
  await visit(built);
  await writeFile(join(scripts, 'static.css'), [...extraStyles].map(([name, style]) => `.${name}{${style}}`).join('\n'));
  await promisify(execFile)(process.execPath, [join(site, 'node_modules/pagefind/lib/runner/bin.cjs'), '--site', built], {
    cwd: site, maxBuffer: 4 * 1024 * 1024,
  });
  // Content + scripts + screenshots + videos share a version, used by the parent SW.
  const hash = createHash('sha256');
  async function fingerprint(directory) {
    for (const entry of (await readdir(directory, { withFileTypes: true })).sort((a,b) => a.name.localeCompare(b.name))) {
      const path = join(directory, entry.name);
      if (entry.isDirectory()) await fingerprint(path);
      else { hash.update(path.slice(built.length).replaceAll('\\', '/')); hash.update(await readFile(path)); }
    }
  }
  await fingerprint(built);
  await writeFile(join(built, 'version.json'), JSON.stringify({ version: hash.digest('hex').slice(0,16) }) + '\n');
  await cp(built, join(output, 'panduan'), { recursive: true });
}
