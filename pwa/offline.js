document.querySelector('[data-pwa-retry]').addEventListener('click', () => location.reload());
addEventListener('online', () => {
  document.querySelector('#pwa-offline-status').textContent = 'Koneksi kembali tersedia. Ketuk Coba lagi untuk membuka EFA.';
});
