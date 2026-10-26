-- name: GetSessionUser :one
SELECT
    u.id AS user_id,
    u.workspace_id,
    u.app,
    u.name,
    u.email,
    u.role,
    s.csrf
FROM public.pblg_sessions s
JOIN public.pblg_users u ON u.id = s.user_id
WHERE s.token_hash = @token_hash
  AND s.expires_at > now()
  AND u.app = @app;

-- name: GetUserForLogin :one
SELECT id, workspace_id, app, name, email, role, password_hash
FROM public.pblg_users
WHERE app = @app AND email = @email;

-- name: CreateSession :exec
INSERT INTO public.pblg_sessions (token_hash, user_id, csrf, expires_at)
VALUES (@token_hash, @user_id, @csrf, @expires_at);

-- name: DeleteSessionByTokenHash :execrows
DELETE FROM public.pblg_sessions
WHERE token_hash = @token_hash;

-- name: GetUserPasswordHash :one
SELECT password_hash
FROM public.pblg_users
WHERE id = @user_id;

-- name: UpdateUserPasswordHash :execrows
UPDATE public.pblg_users
SET password_hash = @password_hash
WHERE id = @user_id;

-- name: DeleteSessionsByUserID :execrows
DELETE FROM public.pblg_sessions
WHERE user_id = @user_id;

-- name: CreateWorkspace :one
INSERT INTO public.pblg_workspaces (id, app, name, code)
VALUES (@id, @app, @name, @code)
RETURNING id, version;

-- name: GetWorkspaceByJoinCode :one
SELECT id
FROM public.pblg_workspaces
WHERE app = @app AND code = @code;

-- name: CreateUser :execrows
INSERT INTO public.pblg_users (id, workspace_id, app, name, email, password_hash, role)
VALUES (@id, @workspace_id, @app, @name, @email, @password_hash, @role)
ON CONFLICT (app, email) DO NOTHING;

-- name: GetWorkspaceForUser :one
SELECT id, name, code, version
FROM public.pblg_workspaces
WHERE id = @workspace_id AND app = @app;

-- name: ListUsersByWorkspace :many
SELECT id, workspace_id, app, name, email, role
FROM public.pblg_users
WHERE workspace_id = @workspace_id AND app = @app
ORDER BY name, id;

-- name: ListAuditEventsByWorkspace :many
SELECT id, actor, action, detail, created_at
FROM public.pblg_audit
WHERE workspace_id = @workspace_id
ORDER BY id DESC
LIMIT @row_limit;

-- name: GetWorkspaceVersion :one
SELECT version
FROM public.pblg_workspaces
WHERE id = @workspace_id;

-- name: InsertAuditEvent :exec
INSERT INTO public.pblg_audit (workspace_id, actor, action, detail)
VALUES (@workspace_id, @actor, @action, @detail);

-- name: BumpWorkspaceVersion :execrows
UPDATE public.pblg_workspaces
SET version = version + 1
WHERE id = @workspace_id;

-- name: LockWorkspaceForUpdate :one
SELECT version
FROM public.pblg_workspaces
WHERE id = @workspace_id AND app = @app
FOR UPDATE;
