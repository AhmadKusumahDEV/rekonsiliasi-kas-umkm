DROP INDEX IF EXISTS public.pblg_audit_entity_idx;

ALTER TABLE public.pblg_audit
    DROP CONSTRAINT IF EXISTS pblg_audit_actor_user_fk;

ALTER TABLE public.pblg_users
    DROP CONSTRAINT IF EXISTS pblg_users_workspace_id_id_key;

ALTER TABLE public.pblg_audit
    DROP CONSTRAINT IF EXISTS pblg_audit_entity_pair;

ALTER TABLE public.pblg_audit
    DROP COLUMN IF EXISTS actor_user_id,
    DROP COLUMN IF EXISTS entity_id,
    DROP COLUMN IF EXISTS entity_type;
