// shared UI checks; the backend still decides whether a transaction is valid.
export function validatePassword(value) {
  if ([...value].length < 12) throw new Error("Kata sandi minimal 12 karakter.");
  if (new TextEncoder().encode(value).length > 72) throw new Error("Kata sandi maksimal 72 byte UTF-8. Huruf non-Latin dan emoji memakai lebih dari satu byte.");
}
export function deduplicatePreview(preview, entries, kind) {
  const existing = new Map(entries.filter(x => x.kind === kind).map(x => [x.source_id, x]));
  const fields = ["source_id","kind","date","reference","name","amount","channel","note"];
  let skipped = Number(preview.skipped || 0);
  const rows = preview.rows.filter(row => {
    const old = existing.get(row.source_id);
    if (!old) return true;
    if (old.canceled || fields.some(field => (old[field] ?? "") !== (row[field] ?? ""))) {
      throw new Error("ID sumber " + row.source_id + " sudah ada dengan data berbeda atau telah dibatalkan.");
    }
    skipped++;
    return false;
  });
  return { ...preview, rows, skipped, count: rows.length, total: rows.reduce((sum,row) => sum + BigInt(row.amount), 0n) };
}
export function selectAllocation(entries, balance, saleID = "", receiptID = "") {
  const sales = entries.filter(x => !x.canceled && x.kind === "sale" && balance(x) > 0);
  const receipts = entries.filter(x => !x.canceled && x.kind === "receipt" && balance(x) > 0);
  const sale = saleID ? sales.find(x => x.id === saleID) : sales[0];
  const receipt = receiptID ? receipts.find(x => x.id === receiptID) : receipts[0];
  if (!sale || !receipt) throw new Error("Transaksi yang dipilih sudah berubah atau tidak tersedia. Muat ulang data dan pilih kembali.");
  return { sale, receipt, sales, receipts };
}
