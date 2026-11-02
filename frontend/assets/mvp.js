// server-owned limits and identifiers; no inferred matches or local policy defaults.
export function importLimits(config) {
  const limits = config?.limits;
  if (!limits || !['max_import_rows','max_csv_bytes','max_entries'].every(key => Number.isSafeInteger(limits[key]) && limits[key] > 0)) {
    throw new Error('Batas impor belum tersedia dari server. Muat ulang konfigurasi sebelum mengimpor.');
  }
  return limits;
}
export function validateImportSize(file, config) {
  const limits = importLimits(config);
  if (!file?.size || file.size > limits.max_csv_bytes) throw new Error(`Pilih CSV berisi data, maksimal ${limits.max_csv_bytes.toLocaleString('id-ID')} byte.`);
  return limits;
}
export function uniqueSuggestion(item, suggestions) {
  if (!item?.suggestion_available || item.status === 'ambiguous' || item.status === 'partial') return null;
  return (suggestions || []).find(match => match.sale.id === item.entry.id || match.receipt.id === item.entry.id) || null;
}
// only group a verified, reciprocal pair; partial and ambiguous entries stay separate.
export function exceptionGroups(items, suggestions) {
  const grouped = new Map();
  const byId = new Map(items.map(item => [item.entry.id, item]));
  return items.reduce((rows, item) => {
    const match = uniqueSuggestion(item, suggestions);
    const counterpart = match && (item.entry.kind === 'sale' ? match.receipt.id : match.sale.id);
    const reciprocal = match && uniqueSuggestion(byId.get(counterpart), suggestions) === match;
    const pairKey = reciprocal ? `${match.sale.id}:${match.receipt.id}` : null;
    if (pairKey && grouped.has(pairKey)) grouped.get(pairKey).items.push(item);
    else {
      const row = {item, match, items: [item], paired: Boolean(pairKey)};
      rows.push(row);
      if (pairKey) grouped.set(pairKey, row);
    }
    return rows;
  }, []);
}
export function auditTarget(event) {
  return ['import','allocation'].includes(event.entity_type) && /^[a-f0-9]{32}$/.test(event.entity_id || '')
    ? `${event.entity_type}:${event.entity_id}` : null;
}
export function mergeAudit(previous, incoming) {
  const seen = new Set(previous.map(row => row.id));
  return [...previous, ...incoming.filter(row => !seen.has(row.id) && seen.add(row.id))];
}
