-- Accounts (internal/auth, internal/account, internal/setup).
-- users has no RLS: it is the access-control layer. The service decides who may read what.

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE lower(username) = lower(sqlc.arg(username));

-- name: CreateUser :one
INSERT INTO users (username, email, display_name, password_hash, instance_role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: SetUserLastLogin :exec
UPDATE users SET last_login_at = now() WHERE id = $1;

-- name: RehashUserPassword :execrows
UPDATE users SET password_hash = sqlc.arg(new_hash)
WHERE id = sqlc.arg(user_id) AND password_hash = sqlc.arg(old_hash) AND status = 'active';

-- TODO(WP-04): ListUsers (cursor), UpdateUser, SetUserPassword, LookupUserByUsername,
--              identities: ListIdentities, GetIdentityByProviderSubject, LinkIdentity, UnlinkIdentity,
--              GetActiveUserByEmail (for oauthAutoLinkByEmail).

-- name: ListIdentityProviders :many
-- For Account.identities.
SELECT provider FROM user_identities WHERE user_id = $1 ORDER BY provider;
