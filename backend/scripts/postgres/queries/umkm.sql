-- name: ListUmkmEntriesByWorkspace :many
SELECT id, workspace_id, batch_id, source_id, kind, occurred_on, reference, name, amount, channel, note, created_at, canceled_at
FROM public.umkm_entries
WHERE workspace_id = @workspace_id
ORDER BY occurred_on, created_at, id;

-- name: ListUmkmImportsByWorkspace :many
SELECT id, workspace_id, kind, checksum, row_count, created_at, canceled_at
FROM public.umkm_imports
WHERE workspace_id = @workspace_id
ORDER BY created_at, id;

-- name: ListUmkmAllocationsByWorkspace :many
SELECT id, workspace_id, sale_entry_id, receipt_entry_id, amount, fee, reason, created_at, reversed_at
FROM public.umkm_allocations
WHERE workspace_id = @workspace_id
ORDER BY created_at, id;

-- name: ListUmkmUnallocatedEntries :many
SELECT id, workspace_id, batch_id, source_id, kind, occurred_on, reference, name, amount, channel, note, created_at, canceled_at
FROM public.umkm_entries e
WHERE e.workspace_id = @workspace_id
  AND e.canceled_at IS NULL
  AND btrim(e.reference) <> ''
  AND NOT EXISTS (
      SELECT 1
      FROM public.umkm_allocations a
      WHERE a.workspace_id = e.workspace_id
        AND a.reversed_at IS NULL
        AND (a.sale_entry_id = e.id OR a.receipt_entry_id = e.id)
  )
ORDER BY e.occurred_on, e.id;

-- name: GetUmkmImportByChecksum :one
SELECT id, workspace_id, kind, checksum, row_count, created_at, canceled_at
FROM public.umkm_imports
WHERE workspace_id = @workspace_id AND checksum = @checksum;

-- name: GetUmkmImportByID :one
SELECT id, workspace_id, kind, checksum, row_count, created_at, canceled_at
FROM public.umkm_imports
WHERE workspace_id = @workspace_id AND id = @id;

-- name: CountUmkmEntries :one
SELECT count(*)::bigint
FROM public.umkm_entries
WHERE workspace_id = @workspace_id;

-- name: ListUmkmEntriesByKindAndSourceIDs :many
SELECT id, workspace_id, batch_id, source_id, kind, occurred_on, reference, name, amount, channel, note, created_at, canceled_at
FROM public.umkm_entries
WHERE workspace_id = @workspace_id
  AND kind = @kind
  AND source_id = ANY(@source_ids::text[])
ORDER BY source_id;

-- name: CreateUmkmImport :one
INSERT INTO public.umkm_imports (id, workspace_id, kind, checksum, row_count)
VALUES (@id, @workspace_id, @kind, @checksum, @row_count)
RETURNING id, workspace_id, kind, checksum, row_count, created_at, canceled_at;

-- name: CreateUmkmEntry :one
INSERT INTO public.umkm_entries (id, workspace_id, batch_id, source_id, kind, occurred_on, reference, name, amount, channel, note)
VALUES (@id, @workspace_id, @batch_id, @source_id, @kind, @occurred_on, @reference, @name, @amount, @channel, @note)
RETURNING id, workspace_id, batch_id, source_id, kind, occurred_on, reference, name, amount, channel, note, created_at, canceled_at;

-- name: GetUmkmAllocationByID :one
SELECT id, workspace_id, sale_entry_id, receipt_entry_id, amount, fee, reason, created_at, reversed_at
FROM public.umkm_allocations
WHERE workspace_id = @workspace_id AND id = @id;

-- name: GetUmkmEntriesByIDs :many
SELECT id, workspace_id, batch_id, source_id, kind, occurred_on, reference, name, amount, channel, note, created_at, canceled_at
FROM public.umkm_entries
WHERE workspace_id = @workspace_id
  AND id = ANY(@ids::text[])
ORDER BY id;

-- name: ListUmkmActiveAllocationsByEntryIDs :many
SELECT id, workspace_id, sale_entry_id, receipt_entry_id, amount, fee, reason, created_at, reversed_at
FROM public.umkm_allocations
WHERE workspace_id = @workspace_id
  AND reversed_at IS NULL
  AND (sale_entry_id = ANY(@entry_ids::text[]) OR receipt_entry_id = ANY(@entry_ids::text[]))
ORDER BY created_at, id;

-- name: CreateUmkmAllocation :one
INSERT INTO public.umkm_allocations (id, workspace_id, sale_entry_id, receipt_entry_id, amount, fee, reason)
VALUES (@id, @workspace_id, @sale_entry_id, @receipt_entry_id, @amount, @fee, @reason)
RETURNING id, workspace_id, sale_entry_id, receipt_entry_id, amount, fee, reason, created_at, reversed_at;

-- name: MarkUmkmAllocationReversed :execrows
UPDATE public.umkm_allocations
SET reversed_at = now()
WHERE workspace_id = @workspace_id AND id = @id AND reversed_at IS NULL;

-- name: CountUmkmActiveAllocationsForImport :one
SELECT count(DISTINCT a.id)::bigint
FROM public.umkm_allocations a
JOIN public.umkm_entries e
  ON e.workspace_id = a.workspace_id
 AND (e.id = a.sale_entry_id OR e.id = a.receipt_entry_id)
WHERE a.workspace_id = @workspace_id
  AND a.reversed_at IS NULL
  AND e.batch_id = @import_id;

-- name: CancelUmkmImport :execrows
UPDATE public.umkm_imports
SET canceled_at = now()
WHERE workspace_id = @workspace_id AND id = @id AND canceled_at IS NULL;

-- name: CancelUmkmImportEntries :execrows
UPDATE public.umkm_entries
SET canceled_at = now()
WHERE workspace_id = @workspace_id AND batch_id = @import_id AND canceled_at IS NULL;
