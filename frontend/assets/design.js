// presentation only; permissions, requests and financial rules stay in app.js / Go.
const escape = (value) =>
  String(value ?? "").replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ],
  );

export function maxyBrand(product, href = "#overview") {
  return `<a class="maxy-brand efa-brand" href="${href}" aria-label="Easy Financial Audit — Beranda"><img src="/assets/efa-mark.svg" width="48" height="48" alt=""><span class="efa-wordmark"><strong>Easy Financial</strong><span>Audit</span></span></a>`;
}
export function shell({ brandName, menu, account, context, nav, footer }) {
  const main = '<main id="main" class="content"></main>';
  return `<div class="shell cash-shell"><header class="cash-header"><div class="cash-brand-row">${menu}${maxyBrand(brandName)}</div>${account}</header><div class="cash-subnav"><div class="context">${context}</div><nav aria-label="Navigasi utama">${nav}</nav></div><div class="workspace">${main}<footer class="cash-footer"><span>Maxy Academy · Proyek PBL</span><span>Data contoh sintetis · Bukan layanan perbankan</span>${footer}</footer></div></div>`;
}
export function dashboard(c) {
  return cashDashboard(c);
}

function dateLabel() {
  return new Intl.DateTimeFormat("id-ID", {
    weekday: "long",
    day: "numeric",
    month: "long",
    year: "numeric",
  }).format(new Date());
}

function heading(c, title, subtitle, action = "") {
  return `<div class="page-title"><div><small class="view-date">${escape(dateLabel())}</small><h1>${escape(title)}</h1><p>${escape(subtitle)}</p></div>${action}</div>`;
}

function cashDashboard(c) {
  const { state, e, money, sumMoney, balance, button, empty } = c;
  const entries = (state.data.state.entries || []).filter((x) => !x.canceled),
    sales = entries.filter((x) => x.kind === "sale"),
    receipts = entries.filter((x) => x.kind === "receipt");
  const open = sales.filter((x) => balance(x) > 0),
    done = sales.length - open.length;
  const rows =
    open
      .slice(0, 6)
      .map(
        (x) =>
          `<div class="cash-transaction"><div><span class="transaction-ref">${e(x.reference || x.source_id)}</span><h3>${e(x.name)}</h3><p>${e(x.channel)} · ${e(x.date)}</p></div><div class="transaction-amount"><strong>${e(money(balance(x)))}</strong><small>Sisa tagihan</small></div>${button("Cocokkan", "allocate", x.id, "small")}</div>`,
      )
      .join("") ||
    empty(
      "Semua penjualan sudah cocok",
      "Tambahkan transaksi periode berikutnya melalui Impor CSV.",
    );
  const channels = [...new Set(receipts.map((x) => x.channel))];
  return (
    heading(
      c,
      `Catatan kas ${state.data.workspace.name}`,
      "Lihat uang yang sudah diterima dan tagihan yang masih perlu dicocokkan.",
      button("Impor CSV", "import", "", "primary"),
    ) +
    `<section class="cash-account"><div class="cash-balance"><span class="account-caption">Tagihan belum cocok</span><strong class="account-amount">${e(money(sumMoney(open, (x) => balance(x))))}</strong><p>${open.length} dari ${sales.length} penjualan masih memiliki sisa.</p><a href="#reconcile" class="button balance-link">Periksa pasangan <span aria-hidden="true">↗</span></a><div class="cash-account-foot"><span>${done} penjualan sudah cocok</span><span>Rupiah / IDR</span></div></div><figure class="cash-business-photo"><img src="/assets/umkm-real.jpg" width="1200" height="800" alt="Foto asli etalase usaha roti"><figcaption><strong>Untuk usaha yang terus berjalan.</strong><span>Foto referensi · Farhad Ibrahimzade / Unsplash</span></figcaption></figure></section>
    <div class="cash-totals"><div><span>Penjualan tercatat</span><strong>${e(money(sumMoney(sales, (x) => x.amount)))}</strong><small>${sales.length} transaksi</small></div><div><span>Penerimaan tercatat</span><strong>${e(money(sumMoney(receipts, (x) => x.amount)))}</strong><small>${receipts.length} transaksi</small></div><div><span>Sudah cocok</span><strong>${done}<small> / ${sales.length}</small></strong><small>Penjualan tanpa sisa tagihan</small></div></div>
    <section class="cash-workspace"><div><div class="section-title"><h2>Perlu dicocokkan</h2><a href="#entries">Semua transaksi ↗</a></div><div class="cash-transactions">${rows}</div></div><aside class="cash-receipt"><div class="receipt-header"><span>${e(state.data.workspace.name)}</span><strong>Penerimaan per kanal</strong></div>${
      channels
        .map(
          (channel) =>
            `<div class="receipt-line"><span>${e(channel)}</span><strong>${e(
              money(
                sumMoney(
                  receipts.filter((x) => x.channel === channel),
                  (x) => x.amount,
                ),
              ),
            )}</strong></div>`,
        )
        .join("") || "<p>Belum ada penerimaan.</p>"
    }<div class="receipt-total"><span>Total diterima</span><strong>${e(money(sumMoney(receipts, (x) => x.amount)))}</strong></div><p>Biaya layanan dan pembayaran sebagian dapat menjelaskan selisih. Saldo penerimaan bukan laba usaha.</p><a href="#reports">Buka rekap kas ↗</a></aside></section>`
  );
}
