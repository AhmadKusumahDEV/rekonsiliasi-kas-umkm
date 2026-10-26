-- Normalized UMKM tables. The PBL baseline keeps its JSONB workspace state for
-- compatibility; these tables hold the data used by the new sqlc query layer.

CREATE TABLE public.umkm_imports (
    id           text PRIMARY KEY CHECK (id ~ '^[0-9a-f]{32}$'),
    workspace_id text NOT NULL REFERENCES public.pblg_workspaces (id) ON DELETE CASCADE,
    kind         text NOT NULL CHECK (kind IN ('sale', 'receipt', 'contoh')),
    checksum     text NOT NULL CHECK (btrim(checksum) <> ''),
    row_count    integer NOT NULL DEFAULT 0 CHECK (row_count BETWEEN 0 AND 500),
    created_at   timestamptz NOT NULL DEFAULT now(),
    canceled_at  timestamptz,
    CONSTRAINT umkm_imports_workspace_id_id_key UNIQUE (workspace_id, id),
    CONSTRAINT umkm_imports_workspace_checksum_key UNIQUE (workspace_id, checksum)
);

CREATE TABLE public.umkm_entries (
    id           text PRIMARY KEY CHECK (id ~ '^[0-9a-f]{32}$'),
    workspace_id text NOT NULL REFERENCES public.pblg_workspaces (id) ON DELETE CASCADE,
    batch_id     text NOT NULL,
    source_id    text NOT NULL CHECK (char_length(btrim(source_id)) BETWEEN 1 AND 80),
    kind         text NOT NULL CHECK (kind IN ('sale', 'receipt')),
    occurred_on  date NOT NULL,
    reference    text NOT NULL DEFAULT '' CHECK (char_length(reference) <= 400),
    name         text NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 400),
    amount       bigint NOT NULL CHECK (amount > 0 AND amount <= 1000000000000),
    channel      text NOT NULL CHECK (char_length(channel) BETWEEN 1 AND 400 AND channel = lower(channel)),
    note         text NOT NULL DEFAULT '' CHECK (char_length(note) <= 400),
    created_at   timestamptz NOT NULL DEFAULT now(),
    canceled_at  timestamptz,
    CONSTRAINT umkm_entries_workspace_id_id_key UNIQUE (workspace_id, id),
    CONSTRAINT umkm_entries_workspace_source_key UNIQUE (workspace_id, kind, source_id),
    CONSTRAINT umkm_entries_import_fk
        FOREIGN KEY (workspace_id, batch_id)
        REFERENCES public.umkm_imports (workspace_id, id)
        ON DELETE CASCADE
);

CREATE TABLE public.umkm_allocations (
    id               text PRIMARY KEY CHECK (id ~ '^[0-9a-f]{32}$'),
    workspace_id     text NOT NULL REFERENCES public.pblg_workspaces (id) ON DELETE CASCADE,
    sale_entry_id    text NOT NULL,
    receipt_entry_id text NOT NULL,
    amount           bigint NOT NULL CHECK (amount > 0 AND amount <= 1000000000000),
    fee              bigint NOT NULL DEFAULT 0 CHECK (fee >= 0 AND fee <= 1000000000000),
    reason           text NOT NULL DEFAULT '' CHECK (char_length(reason) <= 400),
    created_at       timestamptz NOT NULL DEFAULT now(),
    reversed_at      timestamptz,
    CONSTRAINT umkm_allocations_different_entries CHECK (sale_entry_id <> receipt_entry_id),
    CONSTRAINT umkm_allocations_sale_fk
        FOREIGN KEY (workspace_id, sale_entry_id)
        REFERENCES public.umkm_entries (workspace_id, id),
    CONSTRAINT umkm_allocations_receipt_fk
        FOREIGN KEY (workspace_id, receipt_entry_id)
        REFERENCES public.umkm_entries (workspace_id, id)
);

CREATE INDEX umkm_imports_workspace_created_idx
    ON public.umkm_imports (workspace_id, created_at DESC);

CREATE INDEX umkm_entries_workspace_date_idx
    ON public.umkm_entries (workspace_id, occurred_on, id);

CREATE INDEX umkm_entries_match_idx
    ON public.umkm_entries (workspace_id, kind, channel, lower(reference), amount, occurred_on)
    WHERE canceled_at IS NULL;

CREATE INDEX umkm_entries_batch_idx
    ON public.umkm_entries (batch_id);

CREATE INDEX umkm_allocations_active_sale_idx
    ON public.umkm_allocations (workspace_id, sale_entry_id)
    WHERE reversed_at IS NULL;

CREATE INDEX umkm_allocations_active_receipt_idx
    ON public.umkm_allocations (workspace_id, receipt_entry_id)
    WHERE reversed_at IS NULL;

CREATE INDEX umkm_allocations_workspace_created_idx
    ON public.umkm_allocations (workspace_id, created_at DESC);

-- Baseline indexes used by session lookup, cleanup, and member listing.
CREATE INDEX IF NOT EXISTS pblg_sessions_user_idx ON public.pblg_sessions (user_id);
CREATE INDEX IF NOT EXISTS pblg_sessions_expiry_idx ON public.pblg_sessions (expires_at);
CREATE INDEX IF NOT EXISTS pblg_users_workspace_app_idx ON public.pblg_users (workspace_id, app, name);
