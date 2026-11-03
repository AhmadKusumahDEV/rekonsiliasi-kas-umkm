// Public marketing pages only. Never send query strings, fragments, form inputs or account data.
(() => {
  if (location.hostname !== 'efa.madgo.my.id' || location.pathname.startsWith('/app')) return;
  const id = 'G-8YV3VKNXHK';
  const clean = value => { try { const url = new URL(value); return url.origin + url.pathname; } catch { return ''; } };
  window.dataLayer = window.dataLayer || [];
  window.gtag = window.gtag || function () { window.dataLayer.push(arguments); };
  window.gtag('js', new Date());
  window.gtag('config', id, {
    page_location: clean(location.href),
    page_referrer: clean(document.referrer),
    allow_google_signals: false,
    allow_ad_personalization_signals: false
  });
  const script = document.createElement('script');
  script.async = true;
  script.src = 'https://www.googletagmanager.com/gtag/js?id=' + id;
  document.head.append(script);
})();
