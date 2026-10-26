-- Baseline schema imported from Tiga-Prototipe-PBL.
-- Source: internal/platform/store.go (const schema).
--
-- The prototype keeps most domain data inside pblg_workspaces.state (jsonb).
-- IF NOT EXISTS allows this baseline to attach to a database that was already
-- initialized by the prototype without deleting existing data.

CREATE TABLE IF NOT EXISTS public.pblg_workspaces (
    id text PRIMARY KEY,
    app text NOT NULL,
    name text NOT NULL,
    code text NOT NULL UNIQUE,
    version bigint NOT NULL DEFAULT 1,
    state jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.pblg_users (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES public.pblg_workspaces (id) ON DELETE CASCADE,
    app text NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    role text NOT NULL,
    UNIQUE (app, email)
);

CREATE TABLE IF NOT EXISTS public.pblg_sessions (
    token_hash text PRIMARY KEY,
    user_id text NOT NULL REFERENCES public.pblg_users (id) ON DELETE CASCADE,
    csrf text NOT NULL,
    expires_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS public.pblg_jobs (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES public.pblg_workspaces (id) ON DELETE CASCADE,
    app text NOT NULL,
    kind text NOT NULL,
    payload jsonb NOT NULL,
    dedupe text NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0,
    run_at timestamptz NOT NULL DEFAULT now(),
    deadline timestamptz NOT NULL,
    lease_until timestamptz,
    locked_by text,
    last_error text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, dedupe)
);

CREATE INDEX IF NOT EXISTS pblg_jobs_due
    ON public.pblg_jobs (app, status, run_at);

CREATE TABLE IF NOT EXISTS public.pblg_audit (
    id bigserial PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES public.pblg_workspaces (id) ON DELETE CASCADE,
    actor text NOT NULL,
    action text NOT NULL,
    detail text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS pblg_audit_workspace
    ON public.pblg_audit (workspace_id, id DESC);

CREATE TABLE IF NOT EXISTS public.pblg_photos (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES public.pblg_workspaces (id) ON DELETE CASCADE,
    owner_id text NOT NULL REFERENCES public.pblg_users (id) ON DELETE CASCADE,
    mime text NOT NULL,
    body bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
