// build-static replaces this marker whenever public source files change.
const CACHE = 'efa-public-__BUILD_ID__';
const PREFIX = 'efa-public-';
const OFFLINE = '/pwa/offline';
const LIMIT = 60;
const PRECACHE = [
  OFFLINE, '/pwa/pwa.css', '/pwa/offline.js', '/pwa/manifest.webmanifest',
  '/pwa/icons/icon-192.png', '/pwa/icons/icon-512.png',
  '/pwa/icons/maskable-512.png', '/pwa/icons/apple-touch-icon.png',
];
let storageQueue = Promise.resolve();

function publicAsset(url) {
  if (url.pathname === '/pwa/manifest.webmanifest') return true;
  return /^\/(?:assets\/|landing\/assets\/(?:images\/)?|pwa\/(?:icons\/)?)[a-zA-Z0-9_.-]+\.(?:css|js|png|svg|ico|jpe?g|webp|woff2|ttf)$/.test(url.pathname)
    && !url.pathname.includes('..')
    && url.pathname !== '/sw.js';
}
function remember(request, response) {
  const copy = response.clone();
  storageQueue = storageQueue.then(async () => {
    const cache = await caches.open(CACHE);
    await cache.put(request, copy);
    const keys = await cache.keys();
    const disposable = keys.filter(key => !PRECACHE.includes(new URL(key.url).pathname));
    let extra = keys.length - LIMIT;
    for (const key of disposable) {
      if (extra-- <= 0) break;
      await cache.delete(key);
    }
  }).catch(() => { /* quota or private browsing must not block the page. */ });
  return storageQueue;
}
self.addEventListener('install', event => {
  event.waitUntil(caches.open(CACHE).then(cache => cache.addAll(PRECACHE)));
  // allow existing tabs to finish their work; do not force skipWaiting.
});
self.addEventListener('activate', event => {
  event.waitUntil((async () => {
    const names = await caches.keys();
    await Promise.all(names.filter(name => name.startsWith(PREFIX) && name !== CACHE).map(name => caches.delete(name)));
    await self.clients.claim();
  })());
});
self.addEventListener('fetch', event => {
  const request = event.request;
  const url = new URL(request.url);
  // API responses, cookies, account data, exports and writes stay network-only.
  if (request.method !== 'GET' || url.origin !== self.location.origin ||
      url.pathname.startsWith('/api/') || url.pathname.startsWith('/health/')) return;

  if (request.mode === 'navigate') {
    if (!['/', '/index.html', '/app', '/app/', '/app.html', OFFLINE, OFFLINE + '.html'].includes(url.pathname)) return;
    event.respondWith(fetch(request).catch(async () => {
      const cache = await caches.open(CACHE);
      return await cache.match(OFFLINE) || Response.error();
    }));
    return;
  }
  if (!publicAsset(url)) return;
  const network = fetch(request);
  event.waitUntil(network.then(response => {
    if (response.ok && response.type === 'basic' && !response.redirected &&
        !/no-store/i.test(response.headers.get('cache-control') || '')) return remember(request, response);
  }).catch(() => {}));
  event.respondWith(network.catch(async () => {
    const cache = await caches.open(CACHE);
    return await cache.match(request) || Response.error();
  }));
});
