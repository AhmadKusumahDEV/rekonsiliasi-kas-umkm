-- One-time password reset tokens. Only the SHA-256 hash is stored; the raw
-- token exists only in the email link. Requesting a new token invalidates the
-- previous active tokens for the same user.

CREATE TABLE public.pblg_password_reset_tokens (
    token_hash text PRIMARY KEY CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    user_id    text NOT NULL REFERENCES public.pblg_users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX pblg_password_reset_tokens_user_idx
    ON public.pblg_password_reset_tokens (user_id, expires_at DESC);
