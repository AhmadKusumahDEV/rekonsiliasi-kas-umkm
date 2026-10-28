const standalone = () => matchMedia('(display-mode: standalone)').matches || navigator.standalone === true;
let installPrompt;
let help;
let updateNotice;

function syncInstallControls() {
  document.documentElement.classList.toggle('pwa-installed', standalone());
}
addEventListener('beforeinstallprompt', event => {
  event.preventDefault();
  installPrompt = event;
});
addEventListener('appinstalled', () => {
  installPrompt = null;
  document.documentElement.classList.add('pwa-installed');
  if (help?.open) help.close();
});
matchMedia('(display-mode: standalone)').addEventListener('change', syncInstallControls);
syncInstallControls();

function installHelp() {
  if (!help) {
    help = document.createElement('dialog');
    help.className = 'pwa-dialog';
    help.setAttribute('aria-labelledby', 'pwa-help-title');
    help.innerHTML = '<div class="pwa-dialog-head"><h2 id="pwa-help-title">Pasang EFA</h2><button type="button" data-pwa-close aria-label="Tutup petunjuk">×</button></div><p>Buka EFA dari layar utama, tanpa perlu mengetik alamatnya lagi.</p><ol><li><strong>Android · Chrome</strong><span>Buka menu ⋮, lalu pilih Instal aplikasi atau Tambahkan ke layar utama.</span></li><li><strong>iPhone · Safari</strong><span>Ketuk Bagikan, lalu Tambahkan ke Layar Utama. Aktifkan Buka sebagai App jika pilihannya tersedia.</span></li><li><strong>Komputer · Chrome atau Edge</strong><span>Gunakan ikon pemasangan di bilah alamat, atau cari Instal EFA pada menu browser.</span></li></ol><p class="pwa-help-note">Pilihan pemasangan mengikuti browser. Login dan perubahan transaksi tetap memerlukan internet.</p>';
    help.querySelector('[data-pwa-close]').addEventListener('click', () => help.close());
    help.addEventListener('click', event => { if (event.target === help) {
      const box = help.getBoundingClientRect();
      if (event.clientX < box.left || event.clientX > box.right || event.clientY < box.top || event.clientY > box.bottom) help.close();
    } });
    document.body.append(help);
  }
  if (!help.open) help.showModal();
}
document.addEventListener('click', async event => {
  const button = event.target.closest('[data-pwa-install]');
  if (!button) return;
  if (standalone()) return;
  if (!installPrompt) return installHelp();
  const prompt = installPrompt;
  installPrompt = null;
  button.disabled = true;
  try {
    await prompt.prompt();
    await prompt.userChoice;
  } catch { installHelp(); }
  finally { button.disabled = false; }
});

const networkNotice = document.createElement('aside');
networkNotice.className = 'pwa-network';
networkNotice.setAttribute('role', 'status');
networkNotice.textContent = 'Koneksi terputus. Sambungkan internet sebelum menyimpan atau memperbarui transaksi.';
networkNotice.hidden = navigator.onLine;
document.body.append(networkNotice);
addEventListener('offline', () => { networkNotice.hidden = false; });
addEventListener('online', () => { networkNotice.hidden = true; });

function showUpdate() {
  if (updateNotice) return;
  updateNotice = document.createElement('aside');
  updateNotice.className = 'pwa-update';
  updateNotice.setAttribute('role', 'status');
  updateNotice.innerHTML = '<p><strong>Pembaruan EFA siap.</strong> Simpan pekerjaanmu, lalu tutup semua tab dan jendela EFA sebelum membuka kembali.</p><button type="button" aria-label="Tutup pemberitahuan pembaruan">×</button>';
  updateNotice.querySelector('button').addEventListener('click', () => updateNotice.remove());
  document.body.append(updateNotice);
}
async function register() {
  if (!('serviceWorker' in navigator) || !isSecureContext) return;
  try {
    const registration = await navigator.serviceWorker.register('/sw.js', { scope: '/', updateViaCache: 'none' });
    if (registration.waiting && navigator.serviceWorker.controller) showUpdate();
    registration.addEventListener('updatefound', () => {
      const worker = registration.installing;
      worker?.addEventListener('statechange', () => {
        if (worker.state === 'installed' && navigator.serviceWorker.controller) showUpdate();
      });
    });
    addEventListener('online', () => { void registration.update().catch(() => {}); });
  } catch (error) {
    console.warn('PWA belum aktif; website tetap bisa digunakan.', error.name);
  }
}
if (document.readyState === 'complete') void register();
else addEventListener('load', () => { void register(); }, { once: true });
