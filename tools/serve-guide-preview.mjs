// Serve the real build with the same headers as deployment, never proxying API data.
import http from 'node:http';
import { readFile, stat } from 'node:fs/promises';
import { resolve, extname } from 'node:path';
const root = resolve('dist');
const config = JSON.parse(await readFile('vercel.json', 'utf8'));
const types = {'.html':'text/html; charset=utf-8','.js':'text/javascript','.css':'text/css','.svg':'image/svg+xml','.png':'image/png','.jpg':'image/jpeg','.json':'application/json','.wasm':'application/wasm','.mp4':'video/mp4','.vtt':'text/vtt','.csv':'text/csv','.woff2':'font/woff2','.webmanifest':'application/manifest+json'};
const server = http.createServer(async(req,res)=>{
  try {
    const pathname = decodeURIComponent(new URL(req.url, 'http://localhost').pathname);
    let file = resolve(root, '.' + pathname);
    if (file !== root && !file.startsWith(root + (process.platform==='win32'?'\\':'/'))) throw Error('Invalid path');
    if (pathname==='/app') file=resolve(root,'app.html');
    if ((await stat(file).catch(()=>null))?.isDirectory()) file=resolve(file,'index.html');
    const body = await readFile(file);
    const headers = Object.fromEntries(config.headers[0].headers.map(h=>[h.key,h.value]));
    if(pathname.startsWith('/panduan')) {
      const guide = config.headers.find(h=>h.source==='/panduan/(.*)');
      if(guide)for(const h of guide.headers)headers[h.key]=h.value;
      if(process.env.GUIDE_STRICT_SELF==='1')headers['Content-Security-Policy']="default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; media-src 'self'; connect-src 'self'; object-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'";
    }
    res.writeHead(200,{...headers,'Content-Type':types[extname(file)]||'application/octet-stream','Content-Length':body.length});
    res.end(req.method==='HEAD'?undefined:body);
  }catch {res.writeHead(404,{'Content-Type':'text/plain'});res.end('Not found')}
});
server.listen(Number(process.env.PORT||3060),'127.0.0.1',()=>console.log('Guide build: http://127.0.0.1:'+(process.env.PORT||3060)+'/panduan/'));
for(const s of ['SIGINT','SIGTERM'])process.on(s,()=>server.close());
