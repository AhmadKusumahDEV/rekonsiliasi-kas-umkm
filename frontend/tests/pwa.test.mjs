import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';

const source = await readFile(new URL('../../pwa/service-worker.js', import.meta.url), 'utf8');
function harness(fetchImpl) {
  const handlers = {}, stored = new Map(), deleted = [], writes = [];
  const absolute = value => new URL(typeof value === 'string' ? value : value.url, 'https://efa.test').href;
  const cache = {
    addAll: async paths => paths.forEach(path => stored.set(absolute(path), new Response('offline shell'))),
    match: async key => stored.get(absolute(key))?.clone(),
    put: async (key,value) => { writes.push(absolute(key)); stored.set(absolute(key),value); },
    keys: async () => [...stored.keys()].map(url => ({ url })),
    delete: async key => stored.delete(absolute(key)),
  };
  const context = {
    self: { location: { origin:'https://efa.test' }, clients:{ claim:async()=>{} }, addEventListener:(type,fn)=>{handlers[type]=fn;} },
    caches: { open:async()=>cache, keys:async()=>['other-product-cache','efa-public-old','efa-public-__BUILD_ID__'], delete:async name=>deleted.push(name) },
    URL, Response, fetch:fetchImpl,
  };
  vm.runInNewContext(source, context);
  async function dispatch(type, request) {
    const waits = []; let response;
    handlers[type]({ request, waitUntil:promise=>waits.push(promise), respondWith:promise=>{response=promise;} });
    const result = response ? await response : undefined;
    await Promise.all(waits);
    return result;
  }
  const request = (path,mode='cors',method='GET') => ({url:new URL(path,'https://efa.test').href,mode,method});
  return {dispatch,request,stored,writes,deleted};
}
test('PWA never intercepts auth, transactions, SSE, exports, cross-origin traffic or writes', async () => {
  const h = harness(()=>{throw Error('private request reached worker fetch');});
  for (const path of ['/api/v1/auth/me','/api/v1/umkm/state','/api/v1/umkm/events','/api/v1/umkm/export','/health/ready','https://temukas.madgo.my.id/api/v1/auth/me']) {
    assert.equal(await h.dispatch('fetch',h.request(path,'navigate')),undefined);
  }
  assert.equal(await h.dispatch('fetch',h.request('/assets/app.js','cors','POST')),undefined);
  assert.equal(await h.dispatch('fetch',h.request('/private/report.json')),undefined);
  assert.equal(h.stored.size,0);
});
test('offline navigation returns only the public fallback; successful HTML is never cached', async () => {
  const offline = harness(async()=>{throw Error('offline');});
  await offline.dispatch('install');
  const result = await offline.dispatch('fetch',offline.request('/app','navigate'));
  assert.equal(await result.text(),'offline shell');
  assert.equal(await (await offline.dispatch('fetch',offline.request('/pwa/manifest.webmanifest'))).text(),'offline shell');
  assert.equal(offline.stored.has('https://efa.test/app'),false);
  const online = harness(async()=>new Response('<html>current page</html>'));
  assert.match(await (await online.dispatch('fetch',online.request('/app','navigate'))).text(),/current page/);
  assert.equal(online.stored.size,0);
});
test('only valid public assets are cached, stale versions are isolated and cache size stays bounded', async () => {
  const h = harness(async()=>{const response=new Response('public stylesheet');Object.defineProperty(response,'type',{value:'basic'});return response;});
  await h.dispatch('install');
  for(let n=0;n<70;n++) await h.dispatch('fetch',h.request('/assets/style.css?v='+n));
  assert.ok(h.stored.size<=60);
  assert.ok(h.stored.has('https://efa.test/pwa/offline'));
  await h.dispatch('activate');
  assert.deepEqual(h.deleted,['efa-public-old']);
  const bad = harness(async()=>new Response('unavailable',{status:503}));
  await bad.dispatch('fetch',bad.request('/assets/style.css'));
  assert.equal(bad.stored.size,0);
});
