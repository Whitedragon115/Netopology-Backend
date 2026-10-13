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

-- name: ListIdentityProviders :many
-- For Account.identities.
SELECT provider FROM user_identities WHERE user_id = $1 ORDER BY provider;

-- name: ListUsers :many
SELECT * FROM users
WHERE id > sqlc.arg(after)::uuid
  AND (sqlc.arg(search)::text = '' OR username ILIKE '%' || sqlc.arg(search)::text || '%'
       OR display_name ILIKE '%' || sqlc.arg(search)::text || '%')
ORDER BY id LIMIT sqlc.arg(page_limit)::int;

-- name: GetUserForUpdate :one
SELECT * FROM users WHERE id = $1 FOR UPDATE;

-- name: GetInstanceOwnerForUpdate :one
SELECT * FROM users WHERE instance_role = 'owner' FOR UPDATE;

-- name: HasManagerMembership :one
SELECT EXISTS (SELECT 1 FROM workspace_members
  WHERE user_id = $1 AND role IN ('owner', 'admin'));

-- name: LookupUserByUsername :one
SELECT id, username, display_name FROM users
WHERE lower(username) = lower(sqlc.arg(username)) AND status = 'active';

-- name: GetActiveUserByEmail :one
SELECT * FROM users WHERE lower(email) = lower(sqlc.arg(email)) AND status = 'active';

-- name: UpdateUserFields :one
UPDATE users SET
  display_name = CASE WHEN sqlc.arg(set_display_name)::boolean THEN sqlc.arg(display_name)::text ELSE display_name END,
  email = CASE WHEN sqlc.arg(set_email)::boolean THEN sqlc.narg(email)::text ELSE email END,
  status = CASE WHEN sqlc.arg(set_status)::boolean THEN sqlc.arg(status)::text ELSE status END,
  instance_role = CASE WHEN sqlc.arg(set_instance_role)::boolean THEN sqlc.arg(instance_role)::text ELSE instance_role END
WHERE id = sqlc.arg(user_id) RETURNING *;

-- name: DemoteInstanceOwner :exec
UPDATE users SET instance_role = 'admin' WHERE id = $1 AND instance_role = 'owner';

-- name: SetUserPassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1;

-- name: ListIdentities :many
SELECT * FROM user_identities WHERE user_id = $1 ORDER BY created_at, id;

-- name: GetIdentityForUser :one
SELECT * FROM user_identities WHERE id = $1 AND user_id = $2;

-- name: GetIdentityByProviderSubject :one
SELECT * FROM user_identities WHERE provider = $1 AND subject = $2;

-- name: GetIdentityForUserProvider :one
SELECT * FROM user_identities WHERE user_id = $1 AND provider = $2;

-- name: LinkIdentity :one
INSERT INTO user_identities (user_id, provider, subject, email, display_name, last_used_at)
VALUES ($1, $2, $3, $4, $5, now()) RETURNING *;

-- name: TouchIdentity :exec
UPDATE user_identities SET last_used_at = now() WHERE id = $1;

-- name: UnlinkIdentity :execrows
DELETE FROM user_identities WHERE id = $1 AND user_id = $2;

-- name: CountUserIdentities :one
SELECT count(*) FROM user_identities WHERE user_id = $1;

-- name: HasRecentProviderSignIn :one
SELECT EXISTS (
  SELECT 1 FROM sessions s JOIN user_identities i ON i.user_id = s.user_id
  WHERE s.id = sqlc.arg(session_id) AND s.user_id = sqlc.arg(user_id)
    AND s.created_at >= now() - interval '10 minutes'
    AND i.last_used_at >= s.created_at
);
