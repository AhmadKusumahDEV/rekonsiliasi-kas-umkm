-- Remove only rows whose IDs are still present in the legacy JSONB state.
DELETE FROM public.umkm_allocations a
USING public.pblg_workspaces ws
WHERE a.workspace_id = ws.id
  AND ws.app = 'umkm'
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(COALESCE(ws.state->'allocations', '[]'::jsonb)) AS item
      WHERE item->>'id' = a.id
  );

DELETE FROM public.umkm_entries e
USING public.pblg_workspaces ws
WHERE e.workspace_id = ws.id
  AND ws.app = 'umkm'
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(COALESCE(ws.state->'entries', '[]'::jsonb)) AS item
      WHERE item->>'id' = e.id
  );

DELETE FROM public.umkm_imports i
USING public.pblg_workspaces ws
WHERE i.workspace_id = ws.id
  AND ws.app = 'umkm'
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(COALESCE(ws.state->'imports', '[]'::jsonb)) AS item
      WHERE item->>'id' = i.id
  );
