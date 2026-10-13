-- Sessions (internal/auth). The cookie holds a random token; only its SHA-256 is stored.

-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, ip, user_agent, expires_at, remember)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: GetSessionPrincipal :one
-- Resolves a cookie to the signed-in user. Disabled users and expired sessions resolve to nothing.
SELECT s.id AS session_id, s.last_seen_at, s.expires_at, s.remember,
       u.id AS user_id, u.username, u.display_name, u.instance_role
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1
  AND s.expires_at > now()
  AND u.status = 'active';

-- name: TouchSession :exec
-- Sliding expiry. Called at most once per minute per session (see auth.Middleware).
UPDATE sessions SET last_seen_at = now(), expires_at = $2 WHERE id = $1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- name: DeleteUserSessions :execrows
-- Password reset, account disabled: end every session of that user.
DELETE FROM sessions WHERE user_id = $1;

-- name: DeleteOtherUserSessions :execrows
DELETE FROM sessions WHERE user_id = $1 AND id <> $2;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at <= now();
