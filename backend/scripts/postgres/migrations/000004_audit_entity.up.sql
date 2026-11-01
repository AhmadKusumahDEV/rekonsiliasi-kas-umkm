-- Link audit events to the entity they changed without exposing the entity
-- through free-form text. Historical rows keep both columns NULL.

ALTER TABLE public.pblg_audit
    ADD COLUMN entity_type text,
    ADD COLUMN entity_id text,
    ADD COLUMN actor_user_id text;

ALTER TABLE public.pblg_users
    ADD CONSTRAINT pblg_users_workspace_id_id_key UNIQUE (workspace_id, id);

ALTER TABLE public.pblg_audit
    ADD CONSTRAINT pblg_audit_actor_user_fk
        FOREIGN KEY (workspace_id, actor_user_id)
        REFERENCES public.pblg_users (workspace_id, id)
        ON DELETE SET NULL (actor_user_id);

ALTER TABLE public.pblg_audit
    ADD CONSTRAINT pblg_audit_entity_pair CHECK (
        (entity_type IS NULL AND entity_id IS NULL)
        OR (
            entity_type IS NOT NULL
            AND entity_id IS NOT NULL
            AND entity_type IN ('import', 'allocation')
            AND entity_id ~ '^[0-9a-f]{32}$'
        )
    );

CREATE INDEX pblg_audit_entity_idx
    ON public.pblg_audit (workspace_id, entity_type, entity_id, id DESC);
