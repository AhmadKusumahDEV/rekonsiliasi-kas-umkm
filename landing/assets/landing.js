// older shared app links stay usable after the product page takes over the root.
const oldRoutes = new Set(['login','register','overview','entries','reconcile','imports','history','reports','settings']);
if (oldRoutes.has(location.hash.slice(1))) location.replace('/app' + location.hash);

const reduced = matchMedia('(prefers-reduced-motion: reduce)');
const header = document.querySelector('#site-header');
const menu = document.querySelector('.menu-toggle');
const mobileNav = document.querySelector('#mobile-nav');
let scrollFrame;
addEventListener('scroll', () => {
  if (scrollFrame) return;
  scrollFrame = requestAnimationFrame(() => {
    header.dataset.scrolled = String(scrollY > 24);
    scrollFrame = null;
  });
}, { passive: true });
header.dataset.scrolled = String(scrollY > 24);

function closeMenu(returnFocus = false) {
  mobileNav.hidden = true;
  menu.setAttribute('aria-expanded', 'false');
  menu.setAttribute('aria-label', 'Buka menu navigasi');
  header.classList.remove('menu-open');
  if (returnFocus) menu.focus();
}
menu.addEventListener('click', () => {
  const open = menu.getAttribute('aria-expanded') !== 'true';
  mobileNav.hidden = !open;
  menu.setAttribute('aria-expanded', String(open));
  menu.setAttribute('aria-label', open ? 'Tutup menu navigasi' : 'Buka menu navigasi');
  header.classList.toggle('menu-open', open);
});
mobileNav.addEventListener('click', (event) => { if (event.target.closest('a')) closeMenu(); });
document.addEventListener('click', (event) => { if (!header.contains(event.target)) closeMenu(); });
addEventListener('keydown', (event) => {
  document.body.dataset.input = 'keyboard';
  if (event.key === 'Escape' && !mobileNav.hidden) closeMenu(true);
});
addEventListener('pointerdown', () => { document.body.dataset.input = 'pointer'; }, { passive: true });
matchMedia('(min-width: 801px)').addEventListener('change', (event) => { if (event.matches) closeMenu(); });

const hero = document.querySelector('#hero');
const copy = document.querySelector('#hero-copy');
const title = document.querySelector('#hero-title');
const description = document.querySelector('#hero-description');
const pause = document.querySelector('#pause-slider');
const images = [...document.querySelectorAll('[data-slide-image]')];
const tabs = [...document.querySelectorAll('[data-slide]')];
const slides = [
  { title: ['Penjualan tercatat.', 'Uang masuk jelas.'], text: 'Jualan bisa lewat banyak cara. EFA bantu kamu mencocokkan uang yang masuk, satu transaksi demi satu transaksi.' },
  { title: ['Pesanan berjalan.', 'Pembayaran terpantau.'], text: 'Tunai, transfer, atau marketplace. Satukan catatannya dan periksa pembayaran yang masih punya sisa.' },
  { title: ['Beda angka?', 'Telusuri sebabnya.'], text: 'Pembayaran sebagian dan biaya layanan tidak perlu ditebak. Cocokkan dengan catatan, lalu simpan keputusanmu.' },
];
let index = 0;
let timer;
let transition = 0;
let paused = reduced.matches;
let hovered = false;
let visible = true;
let copyAnimation;
const delay = 6500;

function stopTimer() {
  clearTimeout(timer);
  hero.dataset.playing = 'false';
}
function controls() {
  pause.disabled = reduced.matches;
  pause.textContent = reduced.matches ? 'Manual' : paused ? 'Putar ▶' : 'Jeda Ⅱ';
  pause.setAttribute('aria-label', reduced.matches ? 'Mode manual mengikuti pengaturan gerakan' : paused ? 'Putar slide otomatis' : 'Jeda pergantian slide');
}
function schedule() {
  stopTimer();
  controls();
  if (paused || hovered || !visible || document.hidden || reduced.matches) return;
  hero.dataset.playing = 'true';
  timer = setTimeout(() => { void showSlide(index + 1, false); }, delay);
}
async function prepareImage(next) {
  const image = images[next];
  if (!image.src) {
    if (image.dataset.srcset) image.srcset = image.dataset.srcset;
    image.src = image.dataset.src;
  }
  try { await image.decode(); return true; }
  catch { return false; }
}
async function showSlide(next, manual = true) {
  stopTimer();
  const token = ++transition;
  next = (next + slides.length) % slides.length;
  const ready = await prepareImage(next);
  if (token !== transition) return;
  if (ready) images.forEach((image, n) => image.classList.toggle('is-active', n === next));
  index = next;
  title.replaceChildren(document.createTextNode(slides[index].title[0]), document.createElement('br'));
  const accent = document.createElement('span');
  accent.textContent = slides[index].title[1];
  title.append(accent);
  description.textContent = slides[index].text;
  tabs.forEach((button, n) => button.setAttribute('aria-pressed', String(n === index)));
  copyAnimation?.cancel();
  if (!reduced.matches && document.body.dataset.input !== 'keyboard') {
    copyAnimation = copy.animate([
      { opacity: .2, transform: 'translateY(10px)' },
      { opacity: 1, transform: 'translateY(0)' },
    ], { duration: 450, easing: 'cubic-bezier(.22,1,.36,1)' });
  }
  if (manual) document.querySelector('#slide-announcement').textContent = `Slide ${index + 1} dari 3: ${slides[index].title.join(' ')}`;
  schedule();
  void prepareImage((index + 1) % slides.length);
}
tabs.forEach((button, n) => button.addEventListener('click', () => { paused = true; void showSlide(n); }));
document.querySelector('#next-slide').addEventListener('click', () => { paused = true; void showSlide(index + 1); });
document.querySelector('#previous-slide').addEventListener('click', () => { paused = true; void showSlide(index - 1); });
pause.addEventListener('click', () => { paused = !paused; schedule(); });
hero.addEventListener('pointerenter', (event) => { if (event.pointerType === 'mouse') { hovered = true; schedule(); } });
hero.addEventListener('pointerleave', () => { hovered = false; schedule(); });
hero.addEventListener('focusin', (event) => { if (event.target !== pause) paused = true; stopTimer(); controls(); });
hero.addEventListener('focusout', () => { if (!paused) schedule(); });
hero.addEventListener('keydown', (event) => {
  if (!['ArrowRight','ArrowLeft','Home','End'].includes(event.key)) return;
  event.preventDefault();
  paused = true;
  const next = event.key === 'Home' ? 0 : event.key === 'End' ? 2 : index + (event.key === 'ArrowRight' ? 1 : -1);
  void showSlide(next);
});
let touch;
hero.addEventListener('pointerdown', (event) => {
  if (event.pointerType === 'mouse' || !event.isPrimary || event.target.closest('a,button')) return;
  touch = { id: event.pointerId, x: event.clientX, y: event.clientY };
  hero.setPointerCapture(event.pointerId);
});
hero.addEventListener('pointerup', (event) => {
  if (!touch || touch.id !== event.pointerId) return;
  const dx = event.clientX - touch.x, dy = event.clientY - touch.y;
  touch = null;
  if (hero.hasPointerCapture(event.pointerId)) hero.releasePointerCapture(event.pointerId);
  if (Math.abs(dx) > 48 && Math.abs(dx) > Math.abs(dy) * 1.25) { paused = true; void showSlide(index + (dx < 0 ? 1 : -1)); }
});
hero.addEventListener('pointercancel', () => { touch = null; });
document.addEventListener('visibilitychange', schedule);
if ('IntersectionObserver' in window) {
  const heroObserver = new IntersectionObserver(([entry]) => { visible = entry.isIntersecting; schedule(); }, { threshold: .1 });
  heroObserver.observe(hero);
}
reduced.addEventListener('change', () => { paused = true; copyAnimation?.finish(); schedule(); });
schedule();
if ('requestIdleCallback' in window) requestIdleCallback(() => { void prepareImage(1); }, { timeout: 2500 });

const examples = {
  fee: { sale:200000, receipt:190000, fee:10000, caption:'Satu penjualan lewat marketplace', status:'Sudah cocok', explanation:'Uang masuk dan biaya layanan menjelaskan seluruh penjualan.', type:'matched', icon:'✓' },
  partial: { sale:150000, receipt:100000, fee:0, caption:'Pembayaran pertama dari pelanggan', status:'Masih ada sisa', explanation:'Rp50.000 masih perlu ditelusuri. Pastikan apakah pelanggan membayar bertahap.', type:'partial', icon:'↗' },
  unmatched: { sale:100000, receipt:0, fee:0, caption:'Penjualan belum punya pasangan penerimaan', status:'Perlu diperiksa', explanation:'Belum ada pasangan yang ditemukan. Periksa catatan atau tunggu laporan pencairan.', type:'unmatched', icon:'!' },
};
const rupiah = (value) => new Intl.NumberFormat('id-ID', { style:'currency',currency:'IDR',maximumFractionDigits:0 }).format(value);
const exampleTabs = [...document.querySelectorAll('[data-example]')];
function selectExample(button, keyboard = false) {
  const item = examples[button.dataset.example];
  exampleTabs.forEach((tab) => { tab.setAttribute('aria-selected', String(tab === button)); tab.tabIndex = tab === button ? 0 : -1; });
  const panel = document.querySelector('#example-panel');
  panel.setAttribute('aria-labelledby', button.id);
  document.querySelector('#example-caption').textContent = item.caption;
  for (const [id, value] of Object.entries({ sale:item.sale, receipt:item.receipt, 'fee-amount':item.fee, remaining:item.sale-item.receipt-item.fee })) document.querySelector('#example-' + id).textContent = rupiah(value);
  document.querySelector('#example-status').textContent = item.status;
  document.querySelector('#example-explanation').textContent = item.explanation;
  const result = document.querySelector('#example-result');
  result.dataset.status = item.type;
  result.firstElementChild.textContent = item.icon;
  if (!reduced.matches && !keyboard) panel.animate([{opacity:.3,transform:'translateY(5px)'},{opacity:1,transform:'translateY(0)'}],{duration:240,easing:'cubic-bezier(.22,1,.36,1)'});
}
exampleTabs.forEach((button, n) => {
  button.addEventListener('click', () => selectExample(button, document.body.dataset.input === 'keyboard'));
  button.addEventListener('keydown', (event) => {
    if (!['ArrowRight','ArrowLeft','Home','End'].includes(event.key)) return;
    event.preventDefault();
    const next = event.key === 'Home' ? 0 : event.key === 'End' ? 2 : (n + (event.key === 'ArrowRight' ? 1 : -1) + 3) % 3;
    exampleTabs[next].focus();
    selectExample(exampleTabs[next], true);
  });
});

if ('IntersectionObserver' in window && !reduced.matches) {
  const observer = new IntersectionObserver((entries) => {
    for (const entry of entries) if (entry.isIntersecting) { entry.target.classList.remove('waiting'); entry.target.classList.add('is-in-view'); observer.unobserve(entry.target); }
  }, { threshold:.08 });
  document.querySelectorAll('[data-reveal]').forEach((element) => {
    if (element.getBoundingClientRect().top > innerHeight) { element.classList.add('waiting'); observer.observe(element); }
  });
  reduced.addEventListener('change', () => { if (reduced.matches) { observer.disconnect(); document.querySelectorAll('.waiting').forEach((element) => element.classList.remove('waiting')); } });
}
addEventListener('pagehide', () => { stopTimer(); copyAnimation?.cancel(); });
addEventListener('pageshow', schedule);
