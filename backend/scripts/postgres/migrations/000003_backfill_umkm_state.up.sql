-- Copy legacy UMKM JSONB state into the normalized tables. The prototype did
-- not store cancellation/reversal timestamps, so now() is used as the backfill
-- timestamp when a legacy boolean says the row was canceled/reversed.

INSERT INTO public.umkm_imports (id, workspace_id, kind, checksum, row_count, created_at, canceled_at)
SELECT
    item->>'id',
    ws.id,
    item->>'kind',
    item->>'checksum',
    LEAST(500, GREATEST(0, COALESCE((item->>'rows')::integer, 0))),
    COALESCE(NULLIF(item->>'created_at', '')::timestamptz, ws.created_at),
    CASE WHEN COALESCE((item->>'canceled')::boolean, false) THEN now() ELSE NULL END
FROM public.pblg_workspaces ws
CROSS JOIN LATERAL jsonb_array_elements(COALESCE(ws.state->'imports', '[]'::jsonb)) AS item
WHERE ws.app = 'umkm'
  AND item->>'id' ~ '^[0-9a-f]{32}$'
  AND item->>'kind' IN ('sale', 'receipt', 'contoh')
  AND btrim(COALESCE(item->>'checksum', '')) <> ''
ON CONFLICT DO NOTHING;

INSERT INTO public.umkm_entries (
    id, workspace_id, batch_id, source_id, kind, occurred_on, reference, name,
    amount, channel, note, created_at, canceled_at
)
SELECT
    item->>'id',
    ws.id,
    item->>'batch_id',
    item->>'source_id',
    item->>'kind',
    (item->>'date')::date,
    COALESCE(item->>'reference', ''),
    item->>'name',
    (item->>'amount')::bigint,
    lower(item->>'channel'),
    COALESCE(item->>'note', ''),
    imp.created_at,
    CASE WHEN COALESCE((item->>'canceled')::boolean, false) THEN now() ELSE NULL END
FROM public.pblg_workspaces ws
CROSS JOIN LATERAL jsonb_array_elements(COALESCE(ws.state->'entries', '[]'::jsonb)) AS item
JOIN public.umkm_imports imp
  ON imp.workspace_id = ws.id
 AND imp.id = item->>'batch_id'
WHERE ws.app = 'umkm'
  AND item->>'id' ~ '^[0-9a-f]{32}$'
  AND item->>'kind' IN ('sale', 'receipt')
  AND item->>'date' ~ '^\d{4}-\d{2}-\d{2}$'
  AND item->>'amount' ~ '^[0-9]+$'
  AND (item->>'amount')::bigint <= 1000000000000
  AND char_length(btrim(COALESCE(item->>'source_id', ''))) BETWEEN 1 AND 80
  AND char_length(btrim(COALESCE(item->>'name', ''))) BETWEEN 1 AND 400
  AND char_length(COALESCE(item->>'reference', '')) <= 400
  AND char_length(COALESCE(item->>'note', '')) <= 400
  AND char_length(lower(COALESCE(item->>'channel', ''))) BETWEEN 1 AND 400
ON CONFLICT DO NOTHING;

INSERT INTO public.umkm_allocations (
    id, workspace_id, sale_entry_id, receipt_entry_id, amount, fee, reason,
    created_at, reversed_at
)
SELECT
    item->>'id',
    ws.id,
    item->>'sale_id',
    item->>'receipt_id',
    (item->>'amount')::bigint,
    COALESCE((item->>'fee')::bigint, 0),
    COALESCE(item->>'reason', ''),
    COALESCE(NULLIF(item->>'created_at', '')::timestamptz, now()),
    CASE WHEN COALESCE((item->>'reversed')::boolean, false) THEN now() ELSE NULL END
FROM public.pblg_workspaces ws
CROSS JOIN LATERAL jsonb_array_elements(COALESCE(ws.state->'allocations', '[]'::jsonb)) AS item
WHERE ws.app = 'umkm'
  AND item->>'id' ~ '^[0-9a-f]{32}$'
  AND item->>'sale_id' ~ '^[0-9a-f]{32}$'
  AND item->>'receipt_id' ~ '^[0-9a-f]{32}$'
  AND item->>'amount' ~ '^[0-9]+$'
  AND (item->>'amount')::bigint BETWEEN 1 AND 1000000000000
  AND COALESCE(item->>'fee', '0') ~ '^[0-9]+$'
  AND COALESCE((item->>'fee')::bigint, 0) BETWEEN 0 AND 1000000000000
  AND char_length(COALESCE(item->>'reason', '')) <= 400
ON CONFLICT DO NOTHING;
