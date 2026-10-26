DROP INDEX IF EXISTS public.pblg_users_workspace_app_idx;
DROP INDEX IF EXISTS public.pblg_sessions_expiry_idx;
DROP INDEX IF EXISTS public.pblg_sessions_user_idx;

DROP INDEX IF EXISTS public.umkm_allocations_workspace_created_idx;
DROP INDEX IF EXISTS public.umkm_allocations_active_receipt_idx;
DROP INDEX IF EXISTS public.umkm_allocations_active_sale_idx;
DROP INDEX IF EXISTS public.umkm_entries_batch_idx;
DROP INDEX IF EXISTS public.umkm_entries_match_idx;
DROP INDEX IF EXISTS public.umkm_entries_workspace_date_idx;
DROP INDEX IF EXISTS public.umkm_imports_workspace_created_idx;

DROP TABLE IF EXISTS public.umkm_allocations;
DROP TABLE IF EXISTS public.umkm_entries;
DROP TABLE IF EXISTS public.umkm_imports;
