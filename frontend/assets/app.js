import {createImportFlow} from "./import-flow.js?v=handoff1";
import {allocatable,verificationRows,appRoute} from "./verification.js?v=handoff1";
import { importLimits, validateImportSize, uniqueSuggestion, auditTarget, mergeAudit } from "./mvp.js?v=handoff1";
import { apiURL } from "./api-config.js?v=team2";
import { createAPIClient } from "./api-client.js?v=team2";
import { validatePassword, selectAllocation } from "./validation.js?v=handoff1";
import { authScene } from "./auth-scene.js?v=team2";
import { dashboard, maxyBrand, shell } from "./design.js?v=handoff1";

import { presentView, presentAuth } from "./motion.js?v=team2";

const root = document.querySelector("#app");

const modal = document.querySelector("#modal");

const mobileMenu = document.querySelector("#mobile-menu");

const toastElement = document.querySelector("#toast");

const state = {
  config: null,
  connection: "limited",
  audit: { rows: [], loading: false, loaded: false, more: true, error: "", generation: 0 },
  exceptionFilter: "",
  me: null,
  csrf: "",
  data: null,
  source: null,
  list: null,
  page: 1,
  query: "",
  status: "",
  pageSize: 20,
  toastTimer: null,
  refreshing: false,
};

const e = (value) =>
  String(value ?? "").replace(
    /[&<>"']/g,
    (char) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        char
      ],
  );

const money = (value) =>
  new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(typeof value === "bigint" ? value : Number(value || 0));

const sumMoney = (rows, value) =>
  rows.reduce((sum, row) => sum + BigInt(value(row)), 0n);

const date = (value) =>
  value
    ? new Intl.DateTimeFormat("id-ID", {
        day: "numeric",
        month: "short",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      }).format(new Date(value))
    : "Belum ditetapkan";

const key = () => crypto.randomUUID().replaceAll("-", "");

const button = (label, action, id = "", css = "small") =>
  `<button type="button" class="${e(css)}" data-action="${e(action)}" data-id="${e(id)}">${e(label)}</button>`;

const empty = (title, message) =>
  `<div class="empty"><h3>${e(title)}</h3><p>${e(message)}</p></div>`;

const field = (label, name, type = "text", value = "", extra = "") =>
  `<label for="f-${e(name)}">${e(label)}<input id="f-${e(name)}" name="${e(name)}" type="${e(type)}" value="${e(value)}" ${extra}></label>`;

const area = (label, name, value = "", extra = "") =>
  `<label for="f-${e(name)}">${e(label)}<textarea id="f-${e(name)}" name="${e(name)}" ${extra}>${e(value)}</textarea></label>`;

const select = (label, name, options, selected = "") =>
  `<label for="f-${e(name)}">${e(label)}<select id="f-${e(name)}" name="${e(name)}">${options.map(([value, text]) => `<option value="${e(value)}" ${value === selected ? "selected" : ""}>${e(text)}</option>`).join("")}</select></label>`;

const actions = (label) =>
  `<div class="form-actions">${button("Batal", "close", "", "plain")}<button type="submit" class="primary">${e(label)}</button></div>`;

const formError =
  '<div class="error" data-form-error role="alert" tabindex="-1" hidden></div>';

const api = createAPIClient({
  getCSRF: () => state.csrf,
  onUnauthorized() { expireSession(); },
  async onForbidden(path) {
    if (!state.me || path === "/api/v1/auth/me") return;
    const me = await api("/api/v1/auth/me");
    if (me.user.id !== state.me?.id || me.user.workspace_id !== state.me?.workspace_id) {
      expireSession();
      return "Akun di browser berubah. Masuk kembali sebelum melanjutkan.";
    }
    state.csrf = me.csrf;
    return "Sesi diperbarui. Periksa isian, lalu kirim lagi.";
  },
});

const importFlow = createImportFlow({api,getState:()=>state,e,money,refresh});
function normalizeRoute() {
  const raw=location.hash.slice(1).split('?')[0];
  if(['reconcile','exceptions'].includes(raw))history.replaceState(null,'',location.pathname+location.search+'#verifikasi');
  return appRoute(location.hash);
}

function expireSession() {
  if (!state.me) return;
  const dirty = modal.open && modal.dataset.dirty === "true";
  state.me = null;
  state.csrf = "";
  auth();
  toast(dirty ? "Sesi berakhir. Masuk kembali; isian yang belum disimpan perlu diisi ulang." : "Sesi berakhir. Masuk kembali untuk melanjutkan.");
}
function toast(message, persistent = false) {
  clearTimeout(state.toastTimer);
  const text = document.createElement("span");
  text.textContent = message;
  const close = document.createElement("button");
  close.type = "button";
  close.className = "toast-dismiss";
  close.setAttribute("aria-label", "Tutup pesan");
  close.textContent = "×";
  close.addEventListener("click", () => { clearTimeout(state.toastTimer); toastElement.dataset.open = "false"; });
  toastElement.replaceChildren(text, close);
  toastElement.setAttribute("role", persistent ? "alert" : "status");
  toastElement.dataset.open = "true";
  if (!persistent) state.toastTimer = setTimeout(() => {
    toastElement.dataset.open = "false";
  }, 4500);
}

function showError(form, err) {
  if (!form.isConnected) return toast(err.message, true);
  const output = form.querySelector("[data-form-error]");
  if (output) {
    output.hidden = false;
    output.textContent = err.message;
    output.focus();
  } else toast(err.message, true);
}

function stopEvents() {
  state.source?.close();
  state.source = null;
}

function navHTML() {
  const route = normalizeRoute();
  return routes()
    .map(
      ([id, label]) =>
        `<a class="navlink" href="#${e(id)}" title="${e(label)}" ${route === id ? 'aria-current="page"' : ""}><span>${e(label)}</span></a>`,
    )
    .join("");
}

function renderShell() {
  const context = `<strong>${e(state.data.workspace.name)}</strong><small>${e(roles[state.me.role])}</small>`;
  const account = `<div class="account"><span class="connection" role="status" data-online="${state.connection === 'live'}">${connectionLabel()}</span>${button("Data baru · muat ulang", "refresh", "", "small")}<strong>${e(state.me.name)}</strong></div>`;
  root.innerHTML = shell({
    kind: state.config.kind,
    brandName: state.config.brand,
    nav: navHTML(),
    context,
    account,
    menu: button("Menu", "mobile-menu", "", "mobile-toggle plain"),
    footer: button("Keluar akun", "logout", "", "small"),
    workspace: state.data.workspace.name,
  });
  document.querySelector('[data-action="refresh"]').hidden = true;
  renderPage();
}

function title(heading, description, action = "") {
  return `<div class="page-title"><div><h1>${e(heading)}</h1><p>${e(description)}</p></div>${action}</div>`;
}

function metricStrip(items) {
  return `<div class="metrics-wrap"><section class="metrics" aria-label="Ringkasan data">${items.map(([label, value, note]) => `<div class="metric"><p>${e(label)}</p><strong>${e(value)}</strong><small>${e(note)}</small></div>`).join("")}</section></div>`;
}

function rebuildBalances() {
  state.balances = new Map(
    (state.data.state.entries || []).map((x) => [
      x.id,
      x.canceled || x.settled || x.category === "cash" ? 0 : x.amount,
    ]),
  );
  for (const al of state.data.state.allocations || []) {
    if (al.reversed) continue;
    if (state.balances.has(al.sale_id))
      state.balances.set(
        al.sale_id,
        state.balances.get(al.sale_id) - al.amount - al.fee,
      );
    if (state.balances.has(al.receipt_id))
      state.balances.set(
        al.receipt_id,
        state.balances.get(al.receipt_id) - al.amount,
      );
  }
}

function balance(entry) {
  return entry.canceled || entry.settled || entry.category === "cash" ? 0 : (state.balances?.get(entry.id) ?? entry.amount);
}

function listHTML(columns, rows, statusOptions = []) {
  state.list = { columns, rows, statusOptions };
  return `<div class="toolbar"><input type="search" name="search" id="list-search" aria-label="Cari data" placeholder="Cari nama, nomor, atau referensi" value="${e(state.query)}"><div class="inline-actions">${statusOptions.length ? `<select id="list-status" name="status" aria-label="Filter status"><option value="">Semua status</option>${statusOptions.map(([v, t]) => `<option value="${e(v)}" ${state.status === v ? "selected" : ""}>${e(t)}</option>`).join("")}</select>` : ""}<label for="list-size">Tampilkan<select id="list-size" name="page_size">${[20, 50, 100].map((n) => `<option ${state.pageSize === n ? "selected" : ""}>${n}</option>`).join("")}</select></label></div></div><div id="list-results">${listResults()}</div>`;
}

function listResults() {
  const list = state.list;
  if (!list) return "";
  const q = state.query.toLocaleLowerCase("id-ID");
  const filtered = list.rows.filter(
    (x) =>
      (!q || x.search.toLocaleLowerCase("id-ID").includes(q)) &&
      (!state.status || x.status === state.status),
  );
  const max = Math.max(1, Math.ceil(filtered.length / state.pageSize));
  state.page = Math.min(max, Math.max(1, state.page));
  const pageRows = filtered.slice(
    (state.page - 1) * state.pageSize,
    state.page * state.pageSize,
  );
  return `${pageRows.length ? `<div class="table-scroll" role="region" aria-label="Tabel data, geser untuk melihat kolom lain" tabindex="0"><table><thead><tr>${list.columns.map((c) => `<th scope="col">${e(c)}</th>`).join("")}</tr></thead><tbody>${pageRows.map((row) => `<tr>${row.cells.map((cell) => `<td>${cell}</td>`).join("")}</tr>`).join("")}</tbody></table></div>` : empty("Tidak ada data yang cocok", state.query || state.status ? "Ubah pencarian atau filter untuk melihat data lainnya." : "Mulai dengan menambahkan data melalui tombol di halaman ini.")}<div class="countline"><p>${filtered.length} dari ${list.rows.length} data · halaman ${state.page} / ${max}</p><div><button type="button" class="small" data-action="prev" ${state.page <= 1 ? "disabled" : ""}>Sebelumnya</button><button type="button" class="small" data-action="next" ${state.page >= max ? "disabled" : ""}>Berikutnya</button></div></div>`;
}

function entriesPage() {
  const rows = (state.data.state.entries || [])
    .slice()
    .reverse()
    .map((x) => ({
      search: `${x.name} ${x.reference} ${x.source_id} ${x.channel}`,
      status: x.canceled ? "canceled" : x.category === "non_operational" ? "non_operational" : x.kind,
      cells: [
        `<strong>${e(x.reference || x.source_id)}</strong><span class="sub">${e(x.date)}</span>`,
        e(x.name),
        x.kind === "sale" ? "Penjualan" : "Penerimaan",
        `<span class="money">${e(money(x.amount))}</span>`,
        `<span class="money">${x.category==='non_operational'?'—':e(money(balance(x)))}</span>`,
        x.canceled
          ? '<span class="tag">Dibatalkan</span>'
          : x.category === 'non_operational' ? '<span class="tag info">Non-operasional</span>' : x.settled ? '<span class="tag good">Tunai lunas</span>' : balance(x) === 0
            ? '<span class="tag good">Cocok</span>'
            : '<span class="tag warn">Ada sisa</span>',
        x.canceled ? '<span class="muted">Dibatalkan</span>' : x.category === 'non_operational' ? '<span class="muted">Non-operasional</span>' : allocatable(x) && balance(x) > 0
          ? button('Tindak lanjuti', 'entry-followup', x.id, 'small')
          : '<span class="muted">Sudah cocok</span>',
      ],
    }));
  return (
    title(
      "Transaksi",
      "Seluruh transaksi ruang kerja. Ringkasan tidak dibatasi ukuran halaman.",
      button("Impor CSV", "import", "", "primary"),
    ) +
    listHTML(
      ["Referensi", "Nama", "Jenis", "Nominal", "Sisa", "Status", "Tindakan"],
      rows,
      [
        ["sale", "Penjualan"],
        ["receipt", "Penerimaan"],
        ["canceled", "Dibatalkan"],
        ["non_operational", "Non-operasional"],
      ],
    )
  );
}

function allocationHistory() {
  const allocations = (state.data.state.allocations || []).slice().reverse();
  const entries = state.data.state.entries || [];
  const ref = (id) => {
    const x = entries.find((x) => x.id === id);
    return x ? x.reference || x.source_id : "Transaksi";
  };
  return `<section class="allocation-history"><div class="section-title"><h2>Keputusan pencocokan</h2></div>${allocations.length ? `<div class="table-scroll" role="region" aria-label="Tabel data, geser untuk melihat kolom lain" tabindex="0"><table><thead><tr><th>Penjualan / penerimaan</th><th>Dialokasikan</th><th>Biaya</th><th>Catatan</th><th>Status</th><th>Tindakan</th></tr></thead><tbody>${allocations.map((x) => `<tr><td>${e(ref(x.sale_id))}<span class="sub">${e(ref(x.receipt_id))}</span></td><td>${e(money(x.amount))}</td><td>${e(money(x.fee))}</td><td>${e(x.reason || "Tanpa catatan")}</td><td>${x.reversed ? "Dibatalkan" : "Aktif"}</td><td>${x.reversed ? "" : button("Batalkan pasangan", "undo", x.id, "small danger")}</td></tr>`).join("")}</tbody></table></div>` : empty("Belum ada keputusan", "Pasangan yang kamu simpan akan muncul di sini dan dapat dibatalkan.")}</section>`;
}

function importsPage() {
  const rows = (state.data.state.imports || [])
    .slice()
    .reverse()
    .map((x) => ({
      search: `${x.kind} ${x.id}`,
      status: x.canceled ? "canceled" : "active",
      cells: [
        e(date(x.created_at)),
        x.kind === "sale"
          ? "Penjualan"
          : x.kind === "receipt"
            ? "Penerimaan"
            : "Contoh sintetis",
        String(x.rows),
        x.canceled
          ? '<span class="tag">Dibatalkan</span>'
          : '<span class="tag good">Tersimpan</span>',
        !x.canceled
          ? button("Batalkan impor", "cancel-import", x.id, "small danger")
          : "",
      ],
    }));
  return (
    title(
      "Impor CSV",
      "Pratinjau sebelum menyimpan. " + limitDescription(),
      "",
    ) +
    importFlow.html() +
    listHTML(["Waktu impor", "Jenis", "Baris baru", "Status", "Tindakan"], rows)
  );
}

function historyPage() {
  const audit = state.audit;
  return title("Riwayat aktivitas", "Jejak keputusan dari server, termasuk aktivitas yang lebih lama.") +
    (audit.loading ? '<p role="status">Memuat riwayat…</p>' : '') +
    (audit.error ? `<p class="warning" role="alert">${e(audit.error)}</p>${button("Coba muat riwayat", "audit-retry")}` : '') +
    listHTML(["Waktu", "Pelaksana", "Aktivitas", "Keterangan", "Terkait"], audit.rows.map(x => ({
      search: `${x.actor} ${x.action} ${x.detail}`, status: "",
      cells: [e(date(x.created_at)), e(x.actor), `<strong>${e(x.action)}</strong>`, e(x.detail),
        auditTarget(x) ? button("Lihat detail", "audit-entity", auditTarget(x)) : '<span class="muted">Catatan historis<small class="sub">Event lama belum menyimpan tautan ke impor atau alokasi.</small></span>'],
    }))) + (audit.loaded && audit.more && !audit.loading && !audit.error ? button("Muat aktivitas lebih lama", "audit-more") : '') +
    (audit.loaded && !audit.more && !audit.loading ? '<p class="muted">Seluruh riwayat sudah dimuat.</p>' : '');
}

async function loadAudit(reset = false) {
  const audit = state.audit, epoch = state.epoch || 0;
  if (audit.loading && !reset) return;
  const generation = ++audit.generation;
  audit.retryReset = reset;
  audit.loading = true; audit.error = '';
  if (location.hash === '#history') renderPage();
  try {
    const cursor = reset ? null : audit.rows.at(-1)?.id;
    const rows = await api('/api/v1/umkm/audit?limit=50' + (cursor ? '&before_id=' + encodeURIComponent(cursor) : ''));
    if (epoch !== (state.epoch || 0) || !state.me || state.audit !== audit || generation !== audit.generation) return;
    if (!Array.isArray(rows)) throw new Error('Respons riwayat tidak sesuai. Coba lagi.');
    audit.rows = mergeAudit(reset ? [] : audit.rows, rows);
    audit.loaded = true; audit.more = rows.length === 50; audit.version = state.data.workspace.version; audit.retryReset = false;
  } catch (error) {
    if (state.audit !== audit || generation !== audit.generation) return;
    audit.error = error.status === 404 ? 'Riwayat lengkap belum tersedia di server. Catatan lama tetap dipertahankan.' : error.message;
    // Do not substitute the 120-event dashboard snapshot as a complete archive.
    audit.loaded = true;
  } finally {
    if (state.audit === audit && generation === audit.generation) {
      audit.loading = false;
      if (location.hash === '#history' && state.me && canReplacePage()) renderPage();
    }
  }
}

function limitDescription() {
  try {
    const limits = importLimits(state.config);
    return `Maksimal ${limits.max_import_rows.toLocaleString('id-ID')} baris data dan ${limits.max_csv_bytes.toLocaleString('id-ID')} byte per file; ${limits.max_entries.toLocaleString('id-ID')} transaksi per ruang kerja.`;
  } catch { return 'Batas impor belum tersedia dari server; impor dinonaktifkan sampai konfigurasi dimuat ulang.'; }
}

function exceptionsPage() {
 const data=state.data;let content;
 if(data.exceptionsLoading)content='<p role="status">Memuat keputusan yang perlu diperiksa…</p>';
 else if(data.exceptionsError||!Array.isArray(data.exceptions))content='<section class="panel"><p class="warning" role="alert">Antrean belum termuat. Ini bukan berarti semua transaksi sudah cocok.</p>'+button('Coba muat antrean','refresh')+'</section>';
 else{
  const all=verificationRows(data.exceptions,data.suggestions,balance);
  const rows=all.filter(row=>!state.exceptionFilter||row.category===state.exceptionFilter);
  const entries=new Map(data.state.entries.map(x=>[x.id,x]));
  const hasSales=data.state.entries.some(x=>!x.canceled&&x.kind==='sale');
  const hasReceipts=data.state.entries.some(x=>!x.canceled&&x.kind==='receipt'&&x.category!=='non_operational');
  const needsReceipt=data.state.entries.some(x=>allocatable(x)&&x.kind==='sale'&&balance(x)>0);
  const needsSale=data.state.entries.some(x=>allocatable(x)&&x.kind==='receipt'&&balance(x)>0);
  const remaining=new Map(data.exceptions.map(x=>[x.entry.id,x.remaining]));
  const entryHTML=(entry,label)=>`<div class="exception-entry"><small>${e(label)} · ${e(entry.category||'operasional')}</small><strong>${e(entry.reference||entry.source_id)}</strong><span>${e(entry.name)} · ${e(entry.channel)}</span><span>${e(money(entry.amount))} · sisa ${e(money(remaining.get(entry.id)??balance(entry)))}</span></div>`;
  content=((!hasSales&&needsSale)||(!hasReceipts&&needsReceipt)?'<p class="intro-note">Catatan belum lengkap. '+(!hasSales?'Belum ada penjualan.':'Belum ada bukti penerimaan operasional.')+' Data yang diimpor tetap tersimpan; tambahkan sisi lainnya untuk mencocokkan.</p>':'')+`<label for="exception-filter">Tampilkan<select id="exception-filter">${[['','Semua keputusan'],['ready','Siap disetujui'],['ambiguous','Ambigu'],['unmatched','Tanpa kandidat / manual'],['partial','Sisa sebagian']].map(([v,label])=>`<option value="${v}" ${state.exceptionFilter===v?'selected':''}>${label}</option>`).join('')}</select></label><p class="muted">${rows.length} kartu keputusan dari ${all.reduce((n,row)=>n+row.items.length,0)} sisi transaksi. Tunai lunas dan non-operasional tidak memerlukan alokasi.</p>`;
  content+=rows.length?'<div class="exception-list">'+rows.map(row=>{
   const item=row.item, match=row.match;
   const candidates=[...new Set(item.candidate_ids||[])].map(id=>entries.get(id)).filter(allocatable);
   const title=({ready:'Siap diperiksa dan disetujui',ambiguous:'Kandidat perlu dipilih',partial:'Masih ada sisa',unmatched:'Periksa manual'})[row.category];
   return `<article class="panel exception-card"><header><span class="tag ${row.category==='ambiguous'?'warn':'info'}">${title}</span><small>${row.items.length===2?'Satu pasangan':'Satu sisi transaksi'}</small></header><div class="${row.items.length===2?'exception-pair':''}">${row.items.map(x=>entryHTML(x.entry,x.entry.kind==='sale'?'Penjualan':'Penerimaan')).join('')}</div><p>${e(item.reason)}</p>${item.fee_hint?`<div class="intro-note"><strong>Saran biaya ${e(money(item.fee_hint.amount))}</strong><p>${e(item.fee_hint.label)} · ${e(item.fee_hint.source)} · keyakinan ${e(item.fee_hint.confidence)}</p><p>${e(item.fee_hint.reason)}</p><small>Saran perlu diperiksa, belum disetujui.</small></div>`:''}${!match&&candidates.length?'<div class="exception-candidates"><h3>Kandidat pasangan</h3>'+candidates.map(x=>entryHTML(x,'Sisi lawan')).join('')+'</div>':''}<p class="muted">${match?.fee>0?'Pasangan di atas mengikuti saran biaya. ':''}${Number(item.candidate_count)||0} kandidat nominal sama${item.candidate_ids_truncated?'; hanya sebagian ditampilkan. Gunakan pencarian manual untuk kandidat lain.':''}</p><div class="inline-actions">${match?button('Periksa dan setujui','verification-pair',match.sale.id,'secondary'):item.suggestion_available?button('Periksa usulan terbaru','exception-match',item.entry.id,'secondary'):button(item.status==='partial'?'Tambah alokasi':item.status==='ambiguous'?'Pilih kandidat':'Alokasikan manual','exception-choose',item.entry.id,'secondary')}${item.candidate_ids_truncated?button('Cari secara manual','exception-manual',item.entry.id):''}</div></article>`;
  }).join('')+'</div>':empty('Tidak ada keputusan pada tampilan ini','Pasangan yang sudah tersimpan tidak muncul lagi. Periksa filter atau tambahkan catatan yang belum lengkap.');
 }
 const nonop=data.state.entries.filter(x=>!x.canceled&&x.category==='non_operational');
 return title('Verifikasi','Tinjau alasan dan sisa transaksi. Simpan hanya setelah pasangan dan biaya diperiksa.',button('Alokasi manual','allocate','','primary'))+content+`<details class="panel"><summary>Penerimaan non-operasional (${nonop.length})</summary><p>Bunga, administrasi, atau perpindahan dana bukan pembayaran pelanggan. Data tetap tercatat tanpa tombol alokasi.</p>${nonop.map(x=>`<p><strong>${e(x.name)}</strong> · ${e(x.reference||x.source_id)} · ${e(money(x.amount))}</p>`).join('')}</details>`+allocationHistory();
}

function showAuditEntity(target) {
  const [type, id] = target.split(':');
  if (!auditTarget({entity_type:type,entity_id:id})) throw new Error('Tautan riwayat tidak valid.');
  const data = state.data.state;
  const entity = (type === 'import' ? data.imports : data.allocations).find(x => x.id === id);
  if (!entity) throw new Error('Detail terkait tidak ada pada snapshot ini. Muat ulang data; catatan audit tetap tersimpan.');
  const rows = type === 'import'
    ? [['Jenis', entity.kind === 'sale' ? 'Penjualan' : 'Penerimaan'],['Baris baru',entity.rows],['Status',entity.canceled ? 'Dibatalkan' : 'Tersimpan']]
    : [['Penjualan',data.entries.find(x=>x.id === entity.sale_id)?.reference || 'Tidak tersedia'],['Penerimaan',data.entries.find(x=>x.id === entity.receipt_id)?.reference || 'Tidak tersedia'],['Nominal',money(entity.amount)],['Biaya',money(entity.fee)],['Alasan',entity.reason || 'Tidak dicatat'],['Status',entity.reversed ? 'Dibatalkan' : 'Aktif']];
  openModal(type === 'import' ? 'Detail impor terkait' : 'Detail keputusan terkait', '<dl class="details">' + rows.map(([label,value])=>`<dt>${e(label)}</dt><dd>${e(value)}</dd>`).join('') + '</dl>');
}

function settings() {
  return (
    title(
      "Pengaturan akun",
      "Informasi akun dan batas penggunaan ruang kerja.",
    ) +
    `<section class="panel"><h2>${e(state.me.name)}</h2><div class="details"><dl><dt>Email</dt><dd>${e(state.me.email)}</dd><dt>Peran</dt><dd>${e(roles[state.me.role])}</dd><dt>Ruang kerja</dt><dd>${e(state.data.workspace.name)}</dd><dt>Penyimpanan</dt><dd>PostgreSQL, terpisah per aplikasi dan ruang kerja</dd></dl></div><div class="inline-actions">${button("Ganti kata sandi", "password", "", "primary")}${button("Keluar akun", "logout")}<button class="pwa-install" type="button" data-pwa-install>Pasang EFA ↗</button></div><p>Penggantian kata sandi mengakhiri seluruh sesi akun. Data awal adalah contoh sintetis. Jangan unggah data pelanggan nyata sebelum ada izin. ${e(limitDescription())}</p></section>`
    + (state.me.role==='owner'?`<section class="panel danger-zone"><h2>Kosongkan data workspace</h2><p>Hapus transaksi, impor dan alokasi untuk memulai catatan baru. Akun serta audit tetap tersedia. Tindakan ini tidak memiliki pemulihan otomatis.</p>${button('Kosongkan data workspace','clear-data','','danger')}</section>`:'')
  );
}

function showRefreshNotice() {
  const button = document.querySelector('[data-action="refresh"]');
  if (button) button.hidden = false;
}
function canReplacePage() {
  return !modal.open && !importFlow.busy && !document.activeElement?.matches("input,textarea,select");
}
async function refresh(force = false) {
  if (state.refreshTask) {
    try { await state.refreshTask; } catch (error) { if (!force) throw error; }
    if (force && state.me) return refresh();
    return;
  }
  const epoch = state.epoch || 0;
  state.refreshing = true;
  const task = (async () => {
    const auxiliary = Promise.allSettled([
      api("/api/v1/umkm/suggestions"), api("/api/v1/umkm/exceptions"),
    ]);
    const data = await api("/api/v1/umkm/state");
    if (epoch !== (state.epoch || 0) || !state.me) return;
    data.suggestions = [];
    data.suggestionsLoading = true;
    data.exceptions = null;
    data.exceptionsLoading = true;
    state.data = data;
    rebuildBalances();
    if (!document.querySelector(".shell")) renderShell();
    else if (canReplacePage()) renderPage();
    else showRefreshNotice();
    const [suggestionsResult, exceptionsResult] = await auxiliary;
    if (epoch !== (state.epoch || 0) || state.data !== data || !state.me) return;
    const entries = new Map(data.state.entries.map(entry => [entry.id, entry]));
    data.suggestionsLoading = false;
    if (suggestionsResult.status === 'fulfilled' && Array.isArray(suggestionsResult.value)) {
      data.suggestions = suggestionsResult.value.filter(match => [match.sale,match.receipt].every(proposed => {
        const entry = entries.get(proposed.id);
        return entry && !entry.canceled && balance(entry) === entry.amount &&
          ["kind","amount","reference","channel","date"].every(field => entry[field] === proposed[field]);
      }));
    } else data.suggestionsError = true;
    data.exceptionsLoading = false;
    if (exceptionsResult.status === 'fulfilled' && Array.isArray(exceptionsResult.value) &&
      exceptionsResult.value.every(item => {
        const entry = entries.get(item.entry?.id);
        return entry && !entry.canceled && entry.kind === item.entry.kind && balance(entry) === item.remaining;
      })) {
        data.exceptions = exceptionsResult.value.filter(x=>allocatable(x.entry)&&x.remaining>0);
        for(const item of data.exceptions)state.balances.set(item.entry.id,item.remaining);
      }
    else data.exceptionsError = true;
    if (canReplacePage()) renderPage();
    else showRefreshNotice();
    const button = document.querySelector('[data-action="refresh"]');
    if (button && canReplacePage() && !state.pendingVersion && !data.suggestionsError && !data.exceptionsError) button.hidden = true;
    if (data.suggestionsError || data.exceptionsError) showRefreshNotice();
  })();
  state.refreshTask = task;
  try { await task; }
  catch (error) {
    if (epoch === (state.epoch || 0) && state.me) toast(error.message, true);
    throw error;
  } finally {
    if (state.refreshTask === task) {
      state.refreshTask = null;
      state.refreshing = false;
      const pending = state.pendingVersion || 0;
      state.pendingVersion = 0;
      if (state.me && pending > (state.data?.workspace.version || 0)) {
        if (canReplacePage()) queueMicrotask(() => { void refresh().catch(() => {}); });
        else showRefreshNotice();
      }
    }
  }
}
function openModal(heading, content) {
  modal.innerHTML = `<div class="dialog-head"><h2 id="modal-title" tabindex="-1">${e(heading)}</h2>${button("Tutup", "close", "", "small plain")}</div><div class="dialog-body">${content}</div>`;
  modal.dataset.dirty = "false";
  modal.showModal();
  (modal.querySelector('input:not([type="hidden"]),select,textarea') || modal.querySelector("#modal-title")).focus({ preventScroll: true });
}

function closeModal() {
  if (modal.querySelector('form[data-busy="true"]'))
    return toast("Tunggu permintaan selesai sebelum menutup formulir");
  if (
    modal.dataset.dirty === "true" &&
    !confirm("Tutup formulir dan abaikan isian yang belum disimpan?")
  )
    return;
  modal.close();
}

function importModal() { location.hash = "imports"; }

function allocateModal(saleID = "", receiptID = "", options = {}) {
  const entries = state.data.state.entries || [];
  const sales = entries.filter((x) => allocatable(x) && x.kind === "sale" && balance(x) > 0),
    receipts = entries.filter((x) => allocatable(x) && x.kind === "receipt" && balance(x) > 0);
  if (!sales.length || !receipts.length) {
    toast(
      "Tambahkan penjualan dan penerimaan yang masih memiliki sisa terlebih dahulu",
    );
    return;
  }
  const { sale, receipt } = selectAllocation(entries.filter(allocatable), balance, saleID, receiptID);
  openModal(
    "Periksa dan simpan pasangan",
    `<p>${e(options.reason || "Periksa pilihan penjualan dan penerimaan sebelum menyimpan.")}</p><p>Nominal adalah uang penerimaan yang dialokasikan. Biaya mengurangi sisa tagihan, tetapi tidak menambah uang diterima.</p><form data-form="allocate">${formError}<input type="hidden" name="key" value="${key()}">${select(
      "Penjualan",
      "sale_id",
      sales.map((x) => [
        x.id,
        `${x.reference || x.source_id} / ${x.name} / sisa ${money(balance(x))}`,
      ]),
      sale.id,
    )}${select(
      "Penerimaan",
      "receipt_id",
      receipts.map((x) => [
        x.id,
        `${x.reference || x.source_id} / ${x.name} / sisa ${money(balance(x))}`,
      ]),
      receipt.id,
    )}<div class="form-grid">${field("Nominal dialokasikan (Rp)", "amount", "number", Math.min(balance(sale), balance(receipt)), 'required min="1" max="1000000000000" step="1"')}${field("Biaya layanan (Rp)", "fee", "number", 0, 'required min="0" max="1000000000000" step="1"')}</div>${area("Catatan / alasan biaya", "reason", "", 'maxlength="400"')}${actions("Simpan pasangan")}</form>`,
  );
  if (options.choose || !saleID || !receiptID) {
    const names = saleID ? ['receipt_id'] : receiptID ? ['sale_id'] : ['sale_id','receipt_id'];
    for(const name of names){
      const node=modal.querySelector(`[name="${name}"]`);
      if(options.candidates)for(const option of [...node.options])if(!options.candidates.includes(option.value))option.remove();
      node.prepend(new Option('Pilih transaksi terlebih dahulu','',true,true));node.value='';node.required=true;
    }
    modal.querySelector('[name="amount"]').value='';
  }
  if(options.fee>0){
    modal.querySelector('[name="amount"]').value=String(balance(receipt));
    modal.querySelector('[name="fee"]').value=String(options.fee);
    modal.querySelector('[name="reason"]').value=options.reason||'Biaya sesuai saran server; perlu persetujuan';
    const form=modal.querySelector('form');form.dataset.feeSale=sale.id;form.dataset.feeReceipt=receipt.id;
  }
  updateAllocationLimits();
}

function updateAllocationLimits() {
  const form = modal.querySelector('form[data-form="allocate"]');
  if (!form || !state.data) return;
  const entries = state.data.state.entries || [];
  const sale = entries.find(x => x.id === form.elements.sale_id.value);
  const receipt = entries.find(x => x.id === form.elements.receipt_id.value);
  const amount = form.elements.amount, fee = form.elements.fee;
  const saleLeft = sale && !sale.canceled ? balance(sale) : 0;
  const receiptLeft = receipt && !receipt.canceled ? balance(receipt) : 0;
  amount.max = String(Math.max(0, Math.min(saleLeft - Number(fee.value || 0), receiptLeft)));
  fee.max = String(Math.max(0, saleLeft - Number(amount.value || 0)));
  let hint = form.querySelector("#allocation-limit");
  if (!hint) {
    hint = document.createElement("p");
    hint.id = "allocation-limit";
    hint.className = "muted";
    hint.setAttribute("aria-live", "polite");
    form.querySelector(".form-grid").after(hint);
    amount.setAttribute("aria-describedby", hint.id);
    fee.setAttribute("aria-describedby", hint.id);
  }
  hint.textContent = "Sisa tagihan " + money(saleLeft) + " · penerimaan " + money(receiptLeft) + ".";
}

async function init() {
  try {
    state.config = await api("/api/v1/config");
    document.body.dataset.kind = state.config.kind;
    document.title = state.config.brand;
    try {
      const me = await api("/api/v1/auth/me");
      state.me = me.user;
      state.csrf = me.csrf;
    } catch (err) {
      if (err.status !== 401) throw err;
    }
    if (state.me && !["forgot","reset"].includes(normalizeRoute())) {
      startEvents();
      try { await refresh(); }
      catch { showWorkspaceError(); }
    } else { state.me=null; auth(location.hash === "#register"); }
  } catch (err) {
    root.innerHTML = `<main id="main" class="starting"><section class="panel"><h1>Aplikasi belum tersambung</h1><p>${e(err.message)}</p><a class="button primary" href="/app${location.hash === '#register' ? '#register' : '#login'}">Coba sambungkan lagi</a></section></main>`;
  }
}
const roles = { owner: "Pemilik usaha" };
function routes() {
  return [
    ["overview", "Ringkasan"],
    ["entries", "Transaksi"],
    ["verifikasi", "Verifikasi"],
    ["imports", "Impor CSV"],
    ["history", "Riwayat"],
    ["reports", "Laporan"],
    ["settings", "Pengaturan"],
  ];
}
function metrics() {
  const entries = (state.data.state.entries || []).filter((x) => !x.canceled),
    sales = entries.filter((x) => x.kind === "sale"),
    receipts = entries.filter((x) => x.kind === "receipt" && x.category!=="non_operational");
  return metricStrip([
    [
      "Penjualan tercatat",
      money(sumMoney(sales, (x) => x.amount)),
      sales.length + " transaksi",
    ],
    [
      "Penerimaan",
      money(sumMoney(receipts, (x) => x.amount)),
      receipts.length + " transaksi",
    ],
    [
      "Penjualan belum teralokasi",
      money(sumMoney(sales, (x) => balance(x))),
      "Nilai penjualan yang belum punya pasangan",
    ],
    [
      "Sudah cocok",
      sales.filter((x) => balance(x) === 0).length,
      "Penjualan tanpa sisa",
    ],
  ]);
}
function overview() {
  return dashboard({ state, e, money, sumMoney, balance, button, empty });
}
function reportsPage() {
  const allocations = (state.data.state.allocations || []).filter(
    (x) => !x.reversed,
  );
  return (
    title(
      "Laporan",
      "Ringkasan memakai seluruh data usaha.",
      '<a class="button primary" data-action="export" href="' + apiURL("/api/v1/umkm/export") + '">Unduh rekap CSV</a>',
    ) +
    metrics() +
    '<section class="panel"><h2>Keputusan pencocokan</h2><p>' +
    allocations.length +
    " pasangan aktif. Biaya yang sudah dikonfirmasi: " +
    e(money(sumMoney(allocations, (x) => x.fee))) +
    ".</p><p>Penjualan yang belum teralokasi bukan kerugian; bisa berarti belum dibayar, dibayar sebagian, atau menunggu pencairan.</p><p>Ekspor memuat transaksi aktif dan dibatalkan beserta sisa alokasinya. Angka ini bukan perhitungan laba.</p></section>"
  );
}
function showWorkspaceError() {
  if (!state.me) return;
  root.innerHTML = '<main id="main" class="starting"><section class="panel"><h1>Sudah masuk, catatan belum terhubung</h1><p>Sesi akun tetap aktif. Coba muat catatan kembali.</p>' + button("Muat catatan", "refresh", "", "primary") + '</section></main>';
}
async function downloadExport(target) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 32000);
  target.disabled = true;
  target.setAttribute("aria-disabled", "true");
  target.setAttribute("aria-busy", "true");
  try {
    const response = await fetch(apiURL("/api/v1/umkm/export"), { credentials: "include", signal: controller.signal });
    if (!response.ok) {
      if (response.status === 401) expireSession();
      const error = await response.json().catch(() => null);
      throw new Error(error?.message || "Rekap belum dapat diunduh. Coba lagi.");
    }
    if (!(response.headers.get("content-type") || "").includes("text/csv")) throw new Error("Respons rekap tidak sesuai. Coba lagi.");
    const url = URL.createObjectURL(await response.blob());
    const link = document.createElement("a");
    link.href = url;
    link.download = "umkm-rekap.csv";
    document.body.append(link);
    link.click();
    link.remove();
    setTimeout(() => URL.revokeObjectURL(url), 60000);
  } catch (error) {
    toast(error.name === "AbortError" ? "Unduhan belum merespons. Coba lagi." : error instanceof TypeError ? "Koneksi terputus. Sambungkan kembali sebelum mengunduh." : error.message, true);
  } finally {
    clearTimeout(timer);
    target.disabled = false;
    target.removeAttribute("aria-disabled");
    target.removeAttribute("aria-busy");
  }
}
function renderPage() {
  if (!state.data) return;
  state.list = null;
  if (["#login", "#register"].includes(location.hash)) {
    history.replaceState(null, "", location.pathname + location.search + "#overview");
  }
  const route = normalizeRoute(),
    main = document.querySelector("#main");
  if (!main) return;
  const pages = {
    overview,
    entries: entriesPage,
    verifikasi: exceptionsPage,
    imports: importsPage,
    history: historyPage,
    reports: reportsPage,
    settings,
  };
  main.dataset.view = route;
  main.innerHTML = pages[route]
    ? pages[route]()
    : title(
        "Halaman tidak tersedia",
        "Kembali ke ringkasan untuk melanjutkan.",
      ) + '<a class="button" href="#overview">Ringkasan</a>';
  presentView(main, route);
  if (route === "history" && !state.audit.loading && (!state.audit.loaded || (!state.audit.error && state.audit.version !== state.data.workspace.version))) void loadAudit(state.audit.loaded);
  document
    .querySelectorAll("nav")
    .forEach((nav) => (nav.innerHTML = navHTML()));
}
function auth(register = false) {
  state.audit = { rows: [], loading: false, loaded: false, more: true, error: "", generation: (state.audit.generation || 0) + 1 };
  const route = normalizeRoute();
  if (routes().some(([id]) => id === route)) state.returnRoute = route;
  state.epoch = (state.epoch || 0) + 1;
  state.refreshing = false;
  state.refreshTask = null;
  state.pendingVersion = 0;
  importFlow.reset();
  state.data = null;
  state.csrf = "";
  stopEvents();
  // discard protected dialog fields, including passwords, when leaving a session.
  modal.close();
  modal.replaceChildren();
  mobileMenu.close();
  if (["forgot","reset"].includes(route)) return passwordHelp(route);
  state.resetToken = "";
  document.title = (register ? "Daftar" : "Masuk") + " · Easy Financial Audit";
  root.innerHTML =
    '<main id="main" class="auth auth-umkm ' +
    (register ? "is-register" : "") +
    '"><section class="auth-form">' + maxyBrand("Easy Financial Audit", "#login") + '<div class="auth-inner"><a class="auth-home" href="/"><span aria-hidden="true">←</span> Kembali ke beranda</a><div class="auth-heading"><span class="auth-product">Easy Financial Audit</span><small>' +
    (register ? "Mulai catatan usahamu" : "Selamat datang kembali") +
    "</small><h2>" +
    (register ? "Mulai catatan usahamu." : "Selamat datang kembali.") +
    '</h2><p class="muted">' +
    (register
      ? "Satu akun pemilik untuk catatan usaha yang terpisah."
      : "Gunakan email yang terdaftar untuk membuka catatan usahamu.") +
    '</p></div><form data-form="' +
    (register ? "register" : "login") +
    '">' +
    formError +
    (register
      ? field(
          "Nama lengkap",
          "name",
          "text",
          "",
          'required minlength="2" maxlength="100" autocomplete="name"',
        )
      : "") +
    field(
      "Email",
      "email",
      "email",
      "",
      'required maxlength="254" autocomplete="username"',
    ) +
    '<label for="auth-password">Kata sandi<div class="password-field"><input id="auth-password" name="password" type="password" required minlength="12" maxlength="72" autocomplete="' +
    (register ? "new-password" : "current-password") +
    '">' +
    '<button type="button" class="small plain" data-action="toggle-password" aria-label="Tampilkan kata sandi" aria-pressed="false" aria-controls="auth-password">Lihat</button>' +
    "</div>" +
    (register
      ? "<small>Minimal 12 karakter, maksimal 72 byte UTF-8. Emoji dan huruf non-Latin memakai lebih dari satu byte.</small>"
      : "") +
    "</label>" +
    (register
      ? '<div id="owner-fields">' +
        field(
          "Nama usaha",
          "workspace",
          "text",
          "",
          'required minlength="2" maxlength="100"',
        ) +
        "</div>"
      : "") +
    '<button class="primary" type="submit">' +
    (register ? "Daftar akun" : "Masuk") +
    '</button></form>' + (!register ? '<p class="auth-switch"><a href="#forgot">Lupa kata sandi?</a></p>' : '') + '<p class="auth-switch">' +
    (register ? "Sudah punya akun?" : "Belum punya akun?") +
    ' <a href="' +
    (register ? "#login" : "#register") +
    '">' +
    (register ? "Masuk" : "Daftar") +
    '</a></p><button class="pwa-install" type="button" data-pwa-install>Pasang EFA <span aria-hidden="true">↗</span></button></div></section>' + authScene(register) + "</main>";
  presentAuth(root, register);
}
function connectionLabel() {
  return state.connection === 'live' ? 'Data terhubung' : state.connection === 'offline' ? 'Menyambung ulang…' : 'Koneksi terbatas';
}
function updateConnection(status) {
  state.connection = status;
  const indicator = document.querySelector('.connection');
  if (indicator) {
    indicator.dataset.online = String(status === 'live');
    indicator.textContent = connectionLabel();
    indicator.title = status === 'live' ? 'Pembaruan langsung aktif.' : 'Pembaruan langsung terputus. Aplikasi memeriksa data secara berkala.';
  }
}
function startEvents() {
  stopEvents();
  updateConnection('limited');
  let closed = false,
    timer,
    checking = false,
    stream;
  const epoch = state.epoch;
  const current = () => !closed && epoch === state.epoch && state.me;
  const applyVersion = async (version) => {
    if (!current() || version === state.data?.workspace.version) return;
    if (state.refreshing) {
      state.pendingVersion = Math.max(state.pendingVersion || 0, version);
      showRefreshNotice();
      return;
    }
    if (modal.open || document.activeElement?.matches("input,textarea,select")) {
      const button = document.querySelector('[data-action="refresh"]');
      if (button) button.hidden = false;
    } else if (!state.refreshing) await refresh();
  };
  const check = async () => {
    if (!current() || checking || document.visibilityState !== "visible") return;
    checking = true;
    try {
      const next = await api("/api/v1/umkm/state");
      if (current()) await applyVersion(next.workspace.version);
    } catch (err) {
      if (current() && err.status === 401) {
        state.me = null;
        modal.close();
        auth();
      }
    } finally {
      checking = false;
    }
  };
  if ("EventSource" in window) {
    stream = new EventSource(apiURL("/api/v1/umkm/events"), { withCredentials: true });
    stream.addEventListener("state", (event) => {
      const version = Number(event.data);
      if (Number.isSafeInteger(version)) applyVersion(version).catch(() => {});
    });
    // A new stream has no replay guarantee: fetch current state on every open.
    stream.addEventListener("open", () => { if (current()) updateConnection('live'); void check(); });
    stream.addEventListener("error", () => {
      if (!current()) return;
      updateConnection('limited');
      void check();
      clearTimeout(timer);
      timer = setTimeout(poll, 15000);
    });
    stream.addEventListener("resync", () => { void check(); });
    stream.addEventListener("server_close", () => { if (current()) updateConnection('limited'); void check(); });
  }
  state.source = {
    close() {
      closed = true;
      clearTimeout(timer);
      stream?.close();
    },
  };
  const poll = async () => {
    if (closed || !state.me) return;
    let delay = stream?.readyState === 1 ? 60000 : 15000;
    try {
      if (document.visibilityState === "visible" && !state.refreshing) {
        const next = await api("/api/v1/umkm/state");
        if (closed || epoch !== state.epoch) return;
        updateConnection(stream?.readyState === 1 ? 'live' : 'limited');
        if (next.workspace.version !== state.data?.workspace.version) {
          if (
            modal.open ||
            document.activeElement?.matches("input,textarea,select")
          ) {
            const b = document.querySelector('[data-action="refresh"]');
            if (b) b.hidden = false;
          } else await refresh();
        }
      }
    } catch (err) {
      if (closed) return;
      delay = 30000;
      updateConnection('offline');
      if (err.status === 401) {
        stopEvents();
        state.me = null;
        modal.close();
        auth();
        return;
      }
    } finally {
      if (!closed) { clearTimeout(timer); timer = setTimeout(poll, delay); }
    }
  };
  timer = setTimeout(poll, 15000);
}

document.addEventListener("input", (event) => {
  if (modal.contains(event.target)) modal.dataset.dirty = "true";
  if(event.target.closest('form[data-form="clear-data"]'))updateClearButton();
  if (event.target.closest('form[data-form="allocate"]')) updateAllocationLimits();
  if (event.target.id === "list-search") {
    state.query = event.target.value;
    state.page = 1;
    document.querySelector("#list-results").innerHTML = listResults();
  }
});

modal.addEventListener("cancel", (event) => {
  if (modal.querySelector('form[data-busy="true"]')) {
    event.preventDefault();
    toast("Tunggu permintaan selesai sebelum menutup formulir");
    return;
  }
  if (
    modal.dataset.dirty === "true" &&
    !confirm("Tutup formulir dan abaikan isian yang belum disimpan?")
  )
    event.preventDefault();
});
modal.addEventListener("click", (event) => {
  if (event.target !== modal) return;
  const box = modal.getBoundingClientRect();
  if (event.clientX < box.left || event.clientX > box.right || event.clientY < box.top || event.clientY > box.bottom) closeModal();
});

window.addEventListener("hashchange", () => {
  mobileMenu.close();
  state.page = 1;
  state.query = "";
  state.status = "";
  if (["forgot","reset"].includes(normalizeRoute())) {state.me=null;auth();}
  else if (state.me) renderPage();
  else auth(location.hash === "#register");
  const main = document.querySelector("#main");
  if (main) {
    main.tabIndex = -1;
    main.focus({ preventScroll: true });
  }
  window.scrollTo(0, 0);
});

document.querySelector(".skip").addEventListener("click", (event) => {
  event.preventDefault();
  const main = document.querySelector("#main");
  if (main) {
    main.tabIndex = -1;
    main.focus();
    main.scrollIntoView();
  }
});

window.addEventListener("beforeunload", (event) => {
  if ((modal.open && modal.dataset.dirty === "true") || importFlow.dirty) {
    event.preventDefault();
    event.returnValue = "";
  }
});

document.addEventListener("submit", async (event) => {
  const form = event.target.closest("form[data-form]");
  if (!form) return;
  event.preventDefault();
  if (form.dataset.busy === "true") return;
  const button = event.submitter || form.querySelector('button[type="submit"]'),
    label = button?.textContent;
  const fd = new FormData(form),
    get = (name) => String(fd.get(name) || ""),
    action = form.dataset.form;
  const controls = [...form.elements].map((node) => [node, node.disabled]);
  form.dataset.busy = "true";
  form.setAttribute("aria-busy", "true");
  controls.forEach(([node]) => (node.disabled = true));
  if (button) button.textContent = ({login:"Memproses…",register:"Mendaftarkan…","preview-import":"Memeriksa…","flex-import":"Memeriksa…",allocate:"Menyimpan…",password:"Mengganti…"})[action] || "Memproses…";
  try {
    if (action === "login" || action === "register") {
      if (action === "register") validatePassword(get("password"));
      const payload = { email: get("email"), password: get("password") };
      if (action === "register")
        Object.assign(payload, {
          name: get("name"),
          workspace: get("workspace"),
          role: "owner",
        });
      const me = await api("/api/v1/auth/" + action, {
        method: "POST",
        body: payload,
      });
      state.epoch = (state.epoch || 0) + 1;
      state.refreshing = false;
      state.me = me.user;
      state.csrf = me.csrf;
      const destination = action === "login" && routes().some(([id]) => id === state.returnRoute) ? state.returnRoute : "overview";
      state.returnRoute = "";
      history.replaceState(null, "", location.pathname + location.search + "#" + destination);
      startEvents();
      try { await refresh(true); }
      catch { showWorkspaceError(); }
      document.title = state.config.brand;
      return;
    }
    if (action === 'flex-import') { await importFlow.preview(); return; }
    if (action === 'forgot') {
      await api('/api/v1/auth/password/forgot',{method:'POST',body:{email:get('email')}});
      form.innerHTML='<p class="success" role="status">Jika email terdaftar, tautan reset akan dikirim. Periksa inbox dan folder spam.</p><a href="#login">Kembali masuk</a>';return;
    }
    if (action === 'reset') {
      validatePassword(get('password'));
      if(get('password')!==get('confirm_password'))throw Error('Konfirmasi kata sandi belum sama.');
      if(!/^[a-f0-9]{64}$/i.test(state.resetToken||''))throw Error('Tautan reset tidak valid. Minta tautan baru.');
      await api('/api/v1/auth/password/reset',{method:'POST',body:{token:state.resetToken,password:get('password')}});
      state.resetToken='';state.me=null;state.csrf='';history.replaceState(null,'',location.pathname+'#login');auth();toast('Kata sandi berubah. Masuk dengan kata sandi baru.');return;
    }
    if(action==='clear-data'){
      if(get('confirm')!=='HAPUS'||!get('clear_password'))throw Error('Isi kata sandi dan ketik HAPUS.');
      await api('/api/v1/umkm/data',{method:'DELETE',body:{password:get('clear_password'),confirm:get('confirm')}});
      importFlow.reset();state.audit.loaded=false;modal.dataset.dirty='false';modal.close();history.replaceState(null,'',location.pathname+'#overview');await refresh(true);toast('Data dikosongkan. Akun dan riwayat tetap tersedia.');return;
    }
    if (action === "allocate") {
      if (!get("sale_id") || !get("receipt_id")) throw new Error("Pilih penjualan dan penerimaan terlebih dahulu.");
      if (Number(get("fee")) > 0 && get("reason").trim().length < 5) throw new Error("Jelaskan alasan biaya, minimal 5 karakter.");
      updateAllocationLimits();
      const selected = selectAllocation(state.data.state.entries.filter(allocatable), balance, get("sale_id"), get("receipt_id"));
      if (Number(get("amount")) + Number(get("fee")) > balance(selected.sale) || Number(get("amount")) > balance(selected.receipt)) {
        throw new Error("Nominal melebihi sisa terbaru. Periksa kembali pasangan dan biayanya.");
      }
      await api("/api/v1/umkm/allocations", {
        method: "POST",
        body: {
          sale_id: get("sale_id"),
          receipt_id: get("receipt_id"),
          amount: Number(get("amount")),
          fee: Number(get("fee")),
          reason: get("reason"),
          key: get("key"),
        },
      });
    }
    else if (action === "password") {
      validatePassword(get("password"));
      await api("/api/v1/auth/password", {
        method: "PUT",
        body: { current: get("current"), password: get("password") },
      });
      modal.dataset.dirty = "false";
      modal.close();
      state.me = null;
      state.csrf = "";
      auth();
      toast("Kata sandi berubah. Masuk kembali dengan kata sandi baru");
      return;
    } else throw new Error("Formulir tidak dikenal. Muat ulang halaman.");
    modal.dataset.dirty = "false";
    modal.close();
    await refresh(true);
    toast("Pasangan tersimpan");
  } catch (error) {
    if (error.status === 409) await refresh(true).catch(() => {});
    if(action==='clear-data'){form.elements.confirm.value='';form.dataset.confirmAgain='true';}
    if(error.status===429){form.dataset.cooldown='true';setTimeout(()=>{if(form.isConnected){delete form.dataset.cooldown;form.querySelector('button[type=submit]').disabled=false}},60000);}
    showError(form, error);
  } finally {
    form.dataset.busy = "false";
    form.removeAttribute("aria-busy");
    controls.forEach(([node, disabled]) => (node.disabled = disabled));
    if (button?.isConnected) button.textContent = label;
    if(form.dataset.cooldown==='true'&&button?.isConnected)button.disabled=true;
    if(action==='clear-data')updateClearButton();
  }
});
document.addEventListener("click", async (event) => {
  if (event.target.closest("#mobile-menu a")) {
    mobileMenu.close();
    return;
  }
  const target = event.target.closest("[data-action]");
  if (target?.tagName === "A") event.preventDefault();
  if (!target || target.disabled) return;
  const action = target.dataset.action,
    id = target.dataset.id;
  let busyForm, controls;
  try {
    if(action==='flow-save'){await importFlow.commit();return;}
    if(action==='verification-pair'){
      const row=verificationRows(state.data.exceptions||[],state.data.suggestions,balance).find(x=>x.match?.sale.id===id);
      if(!row?.match)throw Error('Usulan berubah. Muat ulang dan periksa kembali.');
      return allocateModal(row.match.sale.id,row.match.receipt.id,{reason:row.match.reason,fee:row.match.fee});
    }
    if(action==='clear-data'){
      if(state.me.role!=='owner')throw Error('Hanya pemilik usaha yang dapat mengosongkan data.');
      openModal('Kosongkan data workspace',`<p class="warning">Transaksi, batch impor, dan alokasi akan dihapus permanen. Akun, workspace dan audit tetap ada. Tidak ada pemulihan otomatis.</p><p>Unduh rekap CSV dari Laporan terlebih dahulu jika diperlukan.</p><form data-form="clear-data">${formError}${field('Kata sandi saat ini','clear_password','password','', 'required autocomplete="current-password"')}${field('Ketik HAPUS untuk mengonfirmasi','confirm','text','', 'required autocomplete="off" pattern="HAPUS"')}<button class="danger" type="submit" disabled>Kosongkan data sekarang</button></form>`);return;
    }
    if (action === "audit-more") return await loadAudit();
    if (action === "audit-retry") return await loadAudit(state.audit.retryReset || !state.audit.rows.length);
    if (action === "audit-entity") return showAuditEntity(id);
    if (["exception-match", "exception-choose", "exception-manual"].includes(action)) {
      const item = state.data.exceptions?.find(x => x.entry.id === id);
      if (!item) throw new Error("Antrean berubah. Muat ulang data terlebih dahulu.");
      if (action === "exception-match") {
        const match = uniqueSuggestion(item, state.data.suggestions);
        if (!match) { await refresh(true); throw new Error("Usulan berubah. Periksa antrean terbaru sebelum melanjutkan."); }
        return allocateModal(match.sale.id, match.receipt.id, { reason: match.reason });
      }
      return allocateModal(item.entry.kind === "sale" ? id : "", item.entry.kind === "receipt" ? id : "", {
        choose: true, reason: item.reason,
        candidates: action === "exception-choose" && item.status === "ambiguous" ? item.candidate_ids || [] : null,
      });
    }
    if (action === "entry-followup") {
      const entry = state.data.state.entries.find(x => x.id === id);
      if (!allocatable(entry) || balance(entry) <= 0) throw new Error("Transaksi sudah berubah. Muat ulang data sebelum melanjutkan.");
      return allocateModal(entry.kind === 'sale' ? id : '', entry.kind === 'receipt' ? id : '', {choose: true});
    }
    if (action === "export") return await downloadExport(target);
    if (action === "close") return closeModal();
    if (action === "toggle-password") {
      const field = document.querySelector("#auth-password");
      field.type = field.type === "password" ? "text" : "password";
      target.textContent = field.type === "password" ? "Lihat" : "Sembunyikan";
      target.setAttribute("aria-pressed", String(field.type === "text"));
      target.setAttribute("aria-label", field.type === "password" ? "Tampilkan kata sandi" : "Sembunyikan kata sandi");
      return;
    }
    if (action === "mobile-menu") {
      mobileMenu.innerHTML =
        '<div class="dialog-body"><div class="mobile-brand">' +
        e(state.config.brand) +
        '</div><nav aria-label="Navigasi mobile">' +
        navHTML() +
        "</nav>" +
        button("Tutup menu", "close-menu") +
        button("Keluar akun", "logout") +
        "</div>";
      mobileMenu.showModal();
      return;
    }
    if (action === "close-menu") return mobileMenu.close();
    if (action === "refresh") return await refresh(true);
    if (action === "logout") {
      await api("/api/v1/auth/logout", { method: "POST", body: {} });
      state.me = null;
      state.csrf = "";
      stopEvents();
      mobileMenu.close();
      history.replaceState(null, "", location.pathname);
      auth();
      return;
    }
    if (action === "import") return importModal();
    if (action === "allocate") return allocateModal(id);
    if (action === "password")
      return openModal(
        "Ganti kata sandi",
        '<form data-form="password">' +
          formError +
          field(
            "Kata sandi saat ini",
            "current",
            "password",
            "",
            'required autocomplete="current-password"',
          ) +
          field(
            "Kata sandi baru",
            "password",
            "password",
            "",
            'required minlength="12" maxlength="72" autocomplete="new-password"',
          ) +
          "<p>Seluruh sesi akun akan diakhiri setelah perubahan.</p>" +
          actions("Ganti kata sandi") +
          "</form>",
      );
    if (["next", "prev"].includes(action)) {
      state.page += action === "next" ? 1 : -1;
      document.querySelector("#list-results").innerHTML = listResults();
      return;
    }
    target.disabled = true;
    if (action === "undo") {
      if (
        !confirm(
          "Batalkan pasangan dan kembalikan sisa saldo? Riwayat tetap tersimpan.",
        )
      )
        return;
      await api("/api/v1/umkm/allocations/" + id + "/undo", {
        method: "POST",
        body: {},
      });
      await refresh(true);
      toast("Pasangan dibatalkan");
    }
    if (action === "cancel-import") {
      if (
        !confirm(
          "Batalkan impor? Pasangan aktif harus dibatalkan terlebih dahulu.",
        )
      )
        return;
      await api("/api/v1/umkm/imports/" + id + "/cancel", {
        method: "POST",
        body: {},
      });
      await refresh(true);
      toast("Impor dibatalkan");
    }
  } catch (error) {
    if (error.status === 409) await refresh(true).catch(() => {});
    if (busyForm?.isConnected) showError(busyForm, error);
    else toast(error.message, true);
  } finally {
    if (busyForm) {
      busyForm.dataset.busy = "false";
      busyForm.removeAttribute("aria-busy");
      controls?.forEach(([node, disabled]) => (node.disabled = disabled));
    }
    if (target.isConnected) target.disabled = false;
  }
});
document.addEventListener("change", (event) => {
  if(event.target.dataset.importKind)importFlow.setFile(event.target.dataset.importKind,event.target.files[0]);
  const allocationForm=event.target.closest('form[data-form="allocate"]');
  if(allocationForm){
    if(allocationForm.dataset.feeSale&&['sale_id','receipt_id'].includes(event.target.name)){
      allocationForm.elements.fee.value='0';allocationForm.elements.amount.value='';delete allocationForm.dataset.feeSale;
    }
    updateAllocationLimits();
  }
  if (event.target.id === "exception-filter") { state.exceptionFilter = event.target.value; renderPage(); return; }
  if (event.target.id === "list-size" || event.target.id === "list-status") {
    state.pageSize = Number(document.querySelector("#list-size").value);
    state.status = document.querySelector("#list-status")?.value || "";
    state.page = 1;
    document.querySelector("#list-results").innerHTML = listResults();
  }
});
function updateClearButton(){
 const form=modal.querySelector('[data-form="clear-data"]');if(!form)return;
 form.querySelector('button[type=submit]').disabled=form.dataset.busy==='true'||form.dataset.cooldown==='true'||!state.me||!form.elements.clear_password.value||form.elements.confirm.value!=='HAPUS';
}
function passwordHelp(route){
 if(route==='reset'){
  const token=new URLSearchParams(location.hash.split('?')[1]||'').get('token');
  if(token!==null){state.resetToken=token;history.replaceState(null,'',location.pathname+'#reset');}
 }else state.resetToken='';
 const valid=/^[a-f0-9]{64}$/i.test(state.resetToken||'');
 document.title=(route==='forgot'?'Lupa kata sandi':'Reset kata sandi')+' · EFA';
 root.innerHTML=`<main id="main" class="auth auth-umkm"><section class="auth-form">${maxyBrand('Easy Financial Audit','#login')}<div class="auth-inner"><a class="auth-home" href="/">Kembali ke beranda</a><h1>${route==='forgot'?'Lupa kata sandi?':'Buat kata sandi baru'}</h1><p>${route==='forgot'?'Masukkan email akunmu. Jika terdaftar, tautan reset akan dikirim.':'Simpan kata sandi baru, lalu masuk kembali.'}</p>${route==='reset'&&!valid?'<p class="error" role="alert">Tautan reset tidak valid atau tidak lengkap. <a href="#forgot">Minta tautan baru</a>.</p>':`<form data-form="${route}">${formError}${route==='forgot'?field('Email','email','email','','required maxlength="254" autocomplete="email"'):field('Kata sandi baru','password','password','','required minlength="12" maxlength="72" autocomplete="new-password"')+field('Ulangi kata sandi baru','confirm_password','password','','required minlength="12" maxlength="72" autocomplete="new-password"')}<button type="submit" class="primary">${route==='forgot'?'Kirim tautan reset':'Simpan kata sandi baru'}</button></form>`}<p class="auth-switch"><a href="#login">Kembali masuk</a>${route==='reset'?' · <a href="#forgot">Minta tautan baru</a>':''}</p></div></section>${authScene(false)}</main>`;
 presentAuth(root,false);
}
init();
