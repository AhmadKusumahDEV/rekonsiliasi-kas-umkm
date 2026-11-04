// Backend categories decide which entries need a human allocation.
export const allocatable = entry => Boolean(entry && !entry.canceled && !entry.settled && entry.category !== 'cash' && entry.category !== 'non_operational');
const normalizeRef = value => String(value || '').toLocaleLowerCase().replace(/[^\p{L}\p{N}]/gu, '');
export function verificationRows(exceptions, suggestions, balance) {
  const items = exceptions.filter(x => allocatable(x.entry) && x.remaining > 0);
  const byId = new Map(items.map(x => [x.entry.id, x]));
  const pairs = new Map();
  for (const match of suggestions || []) {
    if (byId.has(match.sale.id) && byId.has(match.receipt.id)) pairs.set(match.sale.id, { ...match, fee: 0 });
  }
  // Fee hints have no counterpart ID: only offer a pair if both directions are unambiguous.
  const feeChoices = new Map();
  for (const item of items.filter(x => x.entry.kind === 'sale' && x.fee_hint?.amount > 0)) {
    const candidates = items.filter(x => x.entry.kind === 'receipt' && normalizeRef(x.entry.reference) && normalizeRef(x.entry.reference) === normalizeRef(item.entry.reference) && x.remaining + item.fee_hint.amount === item.remaining);
    if (candidates.length === 1) feeChoices.set(item.entry.id, candidates[0]);
  }
  for (const [saleId, receipt] of feeChoices) {
    if ([...feeChoices.values()].filter(x => x.entry.id === receipt.entry.id).length !== 1) continue;
    const sale = byId.get(saleId);
    if (!pairs.has(saleId)) pairs.set(saleId, {sale:sale.entry, receipt:receipt.entry, fee:sale.fee_hint.amount, reason:sale.fee_hint.reason});
  }
  const used = new Set(), rows = [];
  for (const item of items) {
    if (used.has(item.entry.id)) continue;
    let match = [...pairs.values()].find(x => x.sale.id === item.entry.id || x.receipt.id === item.entry.id) || null;
    if (match && (used.has(match.sale.id) || used.has(match.receipt.id))) match = null;
    const own = match ? byId.get(match.sale.id) : item;
    let counterpart = match && byId.get(match.receipt.id);
    if(!match&&item.candidate_ids?.length===1){
      const other=byId.get(item.candidate_ids[0]);
      if(other&&!used.has(other.entry.id)&&other.entry.kind!==item.entry.kind&&other.candidate_ids?.length===1&&other.candidate_ids[0]===item.entry.id)counterpart=other;
    }
    const group = {item:own, match, items:counterpart ? [own,counterpart] : [own], category: match ? 'ready' : item.status === 'partial' ? 'partial' : item.status === 'ambiguous' ? 'ambiguous' : 'unmatched'};
    group.items.forEach(x => used.add(x.entry.id)); rows.push(group);
  }
  return rows;
}
export function appRoute(hash) {
  const route = String(hash || '').replace(/^#/, '').split('?')[0] || 'overview';
  return ['reconcile','exceptions'].includes(route) ? 'verifikasi' : route;
}
