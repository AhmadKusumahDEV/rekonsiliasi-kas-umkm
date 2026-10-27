import { dashboard, maxyBrand, shell } from "./design.js?v=efa2";

import { presentView, presentAuth } from "./motion.js?v=efa2";

const root = document.querySelector("#app");

const modal = document.querySelector("#modal");

const mobileMenu = document.querySelector("#mobile-menu");

const toastElement = document.querySelector("#toast");

const state = {
  config: null,
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

async function api(path, options = {}) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 32000);
  const headers = new Headers(options.headers || {});
  headers.set("X-CSRF-Token", state.csrf);
  let body = options.body;
  if (body && !(body instanceof FormData)) {
    headers.set("Content-Type", "application/json");
    body = JSON.stringify(body);
  }
  try {
    const response = await fetch(path, {
      ...options,
      body,
      headers,
      credentials: "same-origin",
      signal: controller.signal,
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      const err = new Error(
        data.message ||
          data.error ||
          `Permintaan gagal (${response.status}). Coba lagi.`,
      );
      err.status = response.status;
      throw err;
    }
    return data;
  } catch (err) {
    if (err.name === "AbortError")
      throw new Error(
        "Server belum merespons. Data mungkin tersimpan; muat ulang sebelum mengirim lagi.",
      );
    throw err;
  } finally {
    clearTimeout(timeout);
  }
}

function toast(message) {
  clearTimeout(state.toastTimer);
  toastElement.textContent = message;
  toastElement.dataset.open = "true";
  state.toastTimer = setTimeout(() => {
    toastElement.dataset.open = "false";
  }, 4500);
}

function showError(form, err) {
  const output = form.querySelector("[data-form-error]");
  if (output) {
    output.hidden = false;
    output.textContent = err.message;
    output.focus();
  } else toast(err.message);
}

function stopEvents() {
  state.source?.close();
  state.source = null;
}

function navHTML() {
  const route = location.hash.slice(1) || "overview";
  return routes()
    .map(
      ([id, label]) =>
        `<a class="navlink" href="#${e(id)}" title="${e(label)}" ${route === id ? 'aria-current="page"' : ""}><span>${e(label)}</span></a>`,
    )
    .join("");
}

function renderShell() {
  const context = `<strong>${e(state.data.workspace.name)}</strong><small>${e(roles[state.me.role])}</small>`;
  const account = `<div class="account"><span class="connection" data-online="true">Data terhubung</span>${button("Data baru · muat ulang", "refresh", "", "small")}<strong>${e(state.me.name)}</strong></div>`;
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
      x.canceled ? 0 : x.amount,
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
  return entry.canceled ? 0 : (state.balances?.get(entry.id) ?? entry.amount);
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
  return `${pageRows.length ? `<div class="table-scroll"><table><thead><tr>${list.columns.map((c) => `<th scope="col">${e(c)}</th>`).join("")}</tr></thead><tbody>${pageRows.map((row) => `<tr>${row.cells.map((cell) => `<td>${cell}</td>`).join("")}</tr>`).join("")}</tbody></table></div>` : empty("Tidak ada data yang cocok", state.query || state.status ? "Ubah pencarian atau filter untuk melihat data lainnya." : "Mulai dengan menambahkan data melalui tombol di halaman ini.")}<div class="countline"><p>${filtered.length} dari ${list.rows.length} data · halaman ${state.page} / ${max}</p><div><button type="button" class="small" data-action="prev" ${state.page <= 1 ? "disabled" : ""}>Sebelumnya</button><button type="button" class="small" data-action="next" ${state.page >= max ? "disabled" : ""}>Berikutnya</button></div></div>`;
}

function entriesPage() {
  const rows = (state.data.state.entries || [])
    .slice()
    .reverse()
    .map((x) => ({
      search: `${x.name} ${x.reference} ${x.source_id} ${x.channel}`,
      status: x.canceled ? "canceled" : x.kind,
      cells: [
        `<strong>${e(x.reference || x.source_id)}</strong><span class="sub">${e(x.date)}</span>`,
        e(x.name),
        x.kind === "sale" ? "Penjualan" : "Penerimaan",
        `<span class="money">${e(money(x.amount))}</span>`,
        `<span class="money">${e(money(balance(x)))}</span>`,
        x.canceled
          ? '<span class="tag">Dibatalkan</span>'
          : balance(x) === 0
            ? '<span class="tag good">Cocok</span>'
            : '<span class="tag warn">Ada sisa</span>',
      ],
    }));
  return (
    title(
      "Transaksi",
      "Seluruh transaksi ruang kerja. Ringkasan tidak dibatasi ukuran halaman.",
      button("Impor CSV", "import", "", "primary"),
    ) +
    listHTML(
      ["Referensi", "Nama", "Jenis", "Nominal", "Sisa", "Status"],
      rows,
      [
        ["sale", "Penjualan"],
        ["receipt", "Penerimaan"],
        ["canceled", "Dibatalkan"],
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
  return `<section class="allocation-history"><div class="section-title"><h2>Keputusan pencocokan</h2></div>${allocations.length ? `<div class="table-scroll"><table><thead><tr><th>Penjualan / penerimaan</th><th>Dialokasikan</th><th>Biaya</th><th>Catatan</th><th>Status</th><th>Tindakan</th></tr></thead><tbody>${allocations.map((x) => `<tr><td>${e(ref(x.sale_id))}<span class="sub">${e(ref(x.receipt_id))}</span></td><td>${e(money(x.amount))}</td><td>${e(money(x.fee))}</td><td>${e(x.reason || "Tanpa catatan")}</td><td>${x.reversed ? "Dibatalkan" : "Aktif"}</td><td>${x.reversed ? "" : button("Batalkan pasangan", "undo", x.id, "small danger")}</td></tr>`).join("")}</tbody></table></div>` : empty("Belum ada keputusan", "Pasangan yang kamu simpan akan muncul di sini dan dapat dibatalkan.")}</section>`;
}

function matches() {
  const suggestions = state.data.suggestions || [];
  return (
    title(
      "Cocokkan kas",
      "Pasangan unik dari referensi, kanal, nominal, dan rentang tanggal.",
      button("Alokasi manual", "allocate", "", "primary"),
    ) +
    (suggestions.length
      ? `<div class="matches">${suggestions.map((x) => `<article class="match"><div><small>Penjualan ${e(x.sale.reference)}</small><strong>${e(money(x.sale.amount))}</strong><p>${e(x.sale.name)}</p></div><div><small>Penerimaan ${e(x.receipt.channel)}</small><strong>${e(money(x.receipt.amount))}</strong><p>${e(x.receipt.name)}</p></div>${button("Periksa pasangan", "match", x.sale.id, "secondary")}</article>`).join("")}</div>`
      : empty(
          "Tidak ada pasangan unik yang tersisa",
          "Gunakan alokasi manual untuk biaya layanan, pembayaran sebagian, atau pencairan gabungan.",
        )) +
    '<p class="muted">Usulan tidak langsung mengubah saldo. Periksa lalu simpan keputusanmu.</p>' +
    allocationHistory()
  );
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
      "Pratinjau sebelum menyimpan. Maksimal 500 baris dan 1 MB per file.",
      button("Impor CSV", "import", "", "primary"),
    ) +
    `<div class="intro-note"><p>Gunakan satu file penjualan dan satu file penerimaan. Nominal berupa bilangan Rupiah tanpa pemisah ribuan.</p><div class="inline-actions"><a class="button small" href="/assets/contoh-penjualan.csv" download>Contoh penjualan</a><a class="button small" href="/assets/contoh-penerimaan.csv" download>Contoh penerimaan</a></div></div>` +
    listHTML(["Waktu impor", "Jenis", "Baris baru", "Status", "Tindakan"], rows)
  );
}

function historyPage() {
  return (
    title(
      "Riwayat aktivitas",
      "Keputusan dan perubahan tersimpan tanpa menghapus jejak sebelumnya.",
    ) +
    listHTML(
      ["Waktu", "Pelaksana", "Aktivitas", "Keterangan"],
      (state.data.events || []).map((x) => ({
        search: `${x.actor} ${x.action} ${x.detail}`,
        status: "",
        cells: [
          e(date(x.created_at)),
          e(x.actor),
          `<strong>${e(x.action)}</strong>`,
          e(x.detail),
        ],
      })),
    )
  );
}

function settings() {
  return (
    title(
      "Pengaturan akun",
      "Informasi akun dan batas penggunaan ruang kerja.",
    ) +
    `<section class="panel"><h2>${e(state.me.name)}</h2><div class="details"><dl><dt>Email</dt><dd>${e(state.me.email)}</dd><dt>Peran</dt><dd>${e(roles[state.me.role])}</dd><dt>Ruang kerja</dt><dd>${e(state.data.workspace.name)}</dd><dt>Penyimpanan</dt><dd>PostgreSQL, terpisah per aplikasi dan ruang kerja</dd></dl></div><div class="inline-actions">${button("Ganti kata sandi", "password", "", "primary")}${button("Keluar akun", "logout")}</div><p>Penggantian kata sandi mengakhiri seluruh sesi akun. Data awal adalah contoh sintetis. Jangan unggah data pelanggan nyata sebelum ada izin.</p></section>`
  );
}

async function refresh() {
  if (state.refreshing) return;
  state.refreshing = true;
  const epoch = state.epoch || 0;
  try {
    const data = await api("/api/v1/umkm/state");
    if (epoch !== (state.epoch || 0) || !state.me) return;
    data.suggestions = await api("/api/v1/umkm/suggestions");
    if (epoch !== (state.epoch || 0) || !state.me) return;
    state.data = data;
    rebuildBalances();
    if (!document.querySelector(".shell")) renderShell();
    else if (!modal.open) renderPage();
    const btn = document.querySelector('[data-action="refresh"]');
    if (btn) btn.hidden = true;
  } catch (err) {
    if (epoch !== (state.epoch || 0)) return;
    if (err.status === 401) {
      stopEvents();
      state.me = null;
      auth(false);
    } else toast(err.message);
    throw err;
  } finally {
    if (epoch === (state.epoch || 0)) state.refreshing = false;
  }
}

function openModal(heading, content) {
  modal.innerHTML = `<div class="dialog-head"><h2 id="modal-title">${e(heading)}</h2>${button("Tutup", "close", "", "small plain")}</div><div class="dialog-body">${content}</div>`;
  modal.dataset.dirty = "false";
  modal.showModal();
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

function importModal() {
  state.preview = null;
  openModal(
    "Impor transaksi CSV",
    `<p>Pratinjau tidak mengubah saldo. Gunakan ID sumber yang stabil agar impor berulang dapat dikenali.</p><form data-form="preview-import">${formError}${select(
      "Jenis transaksi",
      "kind",
      [
        ["sale", "Penjualan"],
        ["receipt", "Penerimaan"],
      ],
    )}${field("File CSV", "file", "file", "", 'accept=".csv,text/csv" required')}<div class="inline-actions"><a href="/assets/contoh-penjualan.csv" download class="button small">Contoh penjualan</a><a href="/assets/contoh-penerimaan.csv" download class="button small">Contoh penerimaan</a></div><button type="submit" class="primary">Periksa CSV</button></form><div id="import-preview"></div>`,
  );
}

function allocateModal(saleID = "", receiptID = "") {
  const entries = state.data.state.entries || [];
  const sales = entries.filter((x) => x.kind === "sale" && balance(x) > 0),
    receipts = entries.filter((x) => x.kind === "receipt" && balance(x) > 0);
  if (!sales.length || !receipts.length) {
    toast(
      "Tambahkan penjualan dan penerimaan yang masih memiliki sisa terlebih dahulu",
    );
    return;
  }
  saleID ||= sales[0].id;
  receiptID ||= receipts[0].id;
  const sale = sales.find((x) => x.id === saleID) || sales[0],
    receipt = receipts.find((x) => x.id === receiptID) || receipts[0];
  openModal(
    "Periksa dan simpan pasangan",
    `<p>Nominal adalah uang penerimaan yang dialokasikan. Biaya mengurangi sisa tagihan, tetapi tidak menambah uang diterima.</p><form data-form="allocate">${formError}<input type="hidden" name="key" value="${key()}">${select(
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
    if (state.me) {
      await refresh();
      startEvents();
    } else auth(location.hash === "#register");
  } catch (err) {
    root.innerHTML = `<main id="main" class="starting"><section class="panel"><h1>Aplikasi belum tersambung</h1><p>${e(err.message)}</p><a class="button primary" href="/">Muat ulang halaman</a></section></main>`;
  }
}
const roles = { owner: "Pemilik usaha" };
function routes() {
  return [
    ["overview", "Ringkasan"],
    ["entries", "Transaksi"],
    ["reconcile", "Cocokkan kas"],
    ["imports", "Impor CSV"],
    ["history", "Riwayat"],
    ["reports", "Laporan"],
    ["settings", "Pengaturan"],
  ];
}
function metrics() {
  const entries = (state.data.state.entries || []).filter((x) => !x.canceled),
    sales = entries.filter((x) => x.kind === "sale"),
    receipts = entries.filter((x) => x.kind === "receipt");
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
      "Sisa tagihan",
      money(sumMoney(sales, (x) => balance(x))),
      "Perlu ditelusuri",
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
      '<a class="button primary" href="/api/v1/umkm/export" download>Unduh rekap CSV</a>',
    ) +
    metrics() +
    '<section class="panel"><h2>Keputusan pencocokan</h2><p>' +
    allocations.length +
    " pasangan aktif. Biaya yang sudah dikonfirmasi: " +
    e(money(sumMoney(allocations, (x) => x.fee))) +
    ".</p><p>Ekspor memuat transaksi aktif dan dibatalkan beserta sisa alokasinya. Angka ini bukan perhitungan laba.</p></section>"
  );
}
function renderPage() {
  state.list = null;
  const route = location.hash.slice(1) || "overview",
    main = document.querySelector("#main");
  if (!main) return;
  const pages = {
    overview,
    entries: entriesPage,
    reconcile: matches,
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
  document
    .querySelectorAll("nav")
    .forEach((nav) => (nav.innerHTML = navHTML()));
}
function auth(register = false) {
  state.epoch = (state.epoch || 0) + 1;
  state.refreshing = false;
  state.data = null;
  stopEvents();
  document.title = (register ? "Daftar" : "Masuk") + " · Easy Financial Audit";
  root.innerHTML =
    '<main id="main" class="auth auth-umkm ' +
    (register ? "is-register" : "") +
    '"><section class="auth-story">' +
    maxyBrand("Easy Financial Audit", "/") +
    '<img src="/assets/umkm-real.jpg" alt="" width="2400" height="3000"><div class="auth-copy"><h1>Setiap penerimaan, jelas pasangannya.</h1><p>Catatan penjualan, pembayaran masuk, dan selisihnya. Lebih jelas sebelum menutup hari.</p></div><div class="auth-flow" aria-label="Alur rekonsiliasi"><span><b>01</b> Penjualan</span><i aria-hidden="true">→</i><span><b>02</b> Penerimaan</span><i aria-hidden="true">→</i><span><b>03</b> Cocokkan</span></div><small class="muted">Foto referensi · Farhad Ibrahimzade / Unsplash</small></section><section class="auth-form"><div class="auth-inner"><div class="auth-heading"><span class="auth-product">Easy Financial Audit</span><small>' +
    (register ? "Mulai catatan usahamu" : "Selamat datang kembali") +
    "</small><h2>" +
    (register ? "Buat akunmu" : "Masuk ke ruang kerja") +
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
    button("Lihat", "toggle-password", "", "small plain") +
    "</div>" +
    (register
      ? "<small>12–72 karakter. Jangan gunakan kata sandi dari akun lain.</small>"
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
    '</button></form><p class="auth-switch">' +
    (register ? "Sudah punya akun?" : "Belum punya akun?") +
    ' <a href="' +
    (register ? "#login" : "#register") +
    '">' +
    (register ? "Masuk" : "Daftar") +
    "</a></p><small>Prototipe PBL. Jangan unggah data keuangan nyata tanpa izin pemiliknya.</small></div></section></main>";
  presentAuth(root, register);
}
function startEvents() {
  stopEvents();
  let closed = false,
    timer,
    checking = false,
    stream;
  const epoch = state.epoch;
  const current = () => !closed && epoch === state.epoch && state.me;
  const applyVersion = async (version) => {
    if (!current() || version === state.data?.workspace.version) return;
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
    stream = new EventSource("/api/v1/umkm/events");
    stream.addEventListener("state", (event) => {
      const version = Number(event.data);
      if (Number.isSafeInteger(version)) applyVersion(version).catch(() => {});
    });
    // A new stream has no replay guarantee: fetch current state on every open.
    stream.addEventListener("open", () => { void check(); });
    stream.addEventListener("resync", () => { void check(); });
    stream.addEventListener("server_close", () => { void check(); });
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
        const indicator = document.querySelector(".connection");
        if (indicator) {
          indicator.dataset.online = "true";
          indicator.textContent = "Data terhubung";
        }
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
      const indicator = document.querySelector(".connection");
      if (indicator) {
        indicator.dataset.online = "false";
        indicator.textContent = "Menyambung ulang…";
      }
      if (err.status === 401) {
        stopEvents();
        state.me = null;
        modal.close();
        auth();
        return;
      }
    } finally {
      if (!closed) timer = setTimeout(poll, delay);
    }
  };
  timer = setTimeout(poll, 15000);
}

document.addEventListener("input", (event) => {
  if (modal.contains(event.target)) modal.dataset.dirty = "true";
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

window.addEventListener("hashchange", () => {
  mobileMenu.close();
  state.page = 1;
  state.query = "";
  state.status = "";
  if (state.me) renderPage();
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
  if (modal.open && modal.dataset.dirty === "true") {
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
  if (button) button.textContent = "Menyimpan…";
  try {
    if (action === "login" || action === "register") {
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
      if (location.hash) history.replaceState(null, "", "/");
      await refresh();
      document.title = state.config.brand;
      startEvents();
      return;
    }
    if (action === "preview-import") {
      const file = fd.get("file");
      if (!(file instanceof File) || !file.size || file.size > 1048576)
        throw new Error("Pilih CSV berisi data, maksimal 1 MB.");
      const payload = { kind: get("kind"), csv: await file.text() };
      const preview = await api("/api/v1/umkm/imports/preview", {
        method: "POST",
        body: payload,
      });
      state.preview = payload;
      document.querySelector("#import-preview").innerHTML =
        '<div class="success">' +
        preview.count +
        " baris valid / total " +
        e(money(preview.total)) +
        '. Belum disimpan.</div><div class="table-scroll"><table><thead><tr><th>ID</th><th>Nama</th><th>Nominal</th></tr></thead><tbody>' +
        preview.rows
          .slice(0, 5)
          .map(
            (x) =>
              "<tr><td>" +
              e(x.source_id) +
              "</td><td>" +
              e(x.name) +
              "</td><td>" +
              e(money(x.amount)) +
              "</td></tr>",
          )
          .join("") +
        "</tbody></table></div>" +
        buttonHTML("Simpan impor ini", "commit-import", "", "primary");
      return;
    }
    if (action === "allocate")
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
    else if (action === "password") {
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
    await refresh();
    toast("Pasangan tersimpan");
  } catch (error) {
    showError(form, error);
  } finally {
    form.dataset.busy = "false";
    form.removeAttribute("aria-busy");
    controls.forEach(([node, disabled]) => (node.disabled = disabled));
    if (button?.isConnected) button.textContent = label;
  }
});
const buttonHTML = button;
document.addEventListener("click", async (event) => {
  if (event.target.closest("#mobile-menu a")) {
    mobileMenu.close();
    return;
  }
  const target = event.target.closest("[data-action]");
  if (!target || target.disabled) return;
  const action = target.dataset.action,
    id = target.dataset.id;
  let busyForm, controls;
  try {
    if (action === "close") return closeModal();
    if (action === "toggle-password") {
      const field = document.querySelector("#auth-password");
      field.type = field.type === "password" ? "text" : "password";
      target.textContent = field.type === "password" ? "Lihat" : "Sembunyikan";
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
    if (action === "refresh") return await refresh();
    if (action === "logout") {
      await api("/api/v1/auth/logout", { method: "POST", body: {} });
      state.me = null;
      state.csrf = "";
      stopEvents();
      mobileMenu.close();
      history.replaceState(null, "", "/");
      auth();
      return;
    }
    if (action === "import") return importModal();
    if (action === "allocate") return allocateModal(id);
    if (action === "match") {
      const match = state.data.suggestions.find((x) => x.sale.id === id);
      if (!match) return toast("Usulan berubah. Muat ulang data.");
      return allocateModal(match.sale.id, match.receipt.id);
    }
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
    if (action === "commit-import") {
      if (!state.preview) throw new Error("Periksa CSV terlebih dahulu.");
      busyForm = modal.querySelector("form");
      if (busyForm.dataset.busy === "true") return;
      controls = [...busyForm.elements].map((node) => [node, node.disabled]);
      controls.forEach(([node]) => (node.disabled = true));
      busyForm.dataset.busy = "true";
      busyForm.setAttribute("aria-busy", "true");
      const result = await api("/api/v1/umkm/imports", {
        method: "POST",
        body: state.preview,
      });
      state.preview = null;
      modal.dataset.dirty = "false";
      modal.close();
      await refresh();
      toast(
        result.duplicate
          ? "File sudah pernah diimpor. Tidak ada data ganda"
          : result.imported +
              " baris baru tersimpan, " +
              result.skipped +
              " baris identik dilewati",
      );
    }
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
      await refresh();
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
      await refresh();
      toast("Impor dibatalkan");
    }
  } catch (error) {
    if (busyForm?.isConnected) showError(busyForm, error);
    else toast(error.message);
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
  if (event.target.closest('[data-form="preview-import"]')) {
    state.preview = null;
    document.querySelector("#import-preview").replaceChildren();
  }
  if (event.target.id === "list-size" || event.target.id === "list-status") {
    state.pageSize = Number(document.querySelector("#list-size").value);
    state.status = document.querySelector("#list-status")?.value || "";
    state.page = 1;
    document.querySelector("#list-results").innerHTML = listResults();
  }
});
init();
