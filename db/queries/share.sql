-- name: ListShareLinks :many
SELECT * FROM share_links WHERE workspace_id=$1 AND canvas_id=$2 ORDER BY created_at DESC, id DESC;

-- name: CreateShareLink :one
INSERT INTO share_links (workspace_id,canvas_id,label,token_hash,password_hash,expires_at,created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING *;

-- name: GetShareLink :one
SELECT * FROM share_links WHERE workspace_id=$1 AND id=$2 FOR UPDATE;

-- name: GetActiveShareByToken :one
SELECT * FROM share_links WHERE token_hash=$1 AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at>now());

-- name: LockActiveShareLink :one
SELECT * FROM share_links WHERE workspace_id=$1 AND id=$2 AND revoked_at IS NULL
AND (expires_at IS NULL OR expires_at>now()) FOR SHARE;

-- name: UpdateShareLink :one
UPDATE share_links SET label=$3,password_hash=$4,expires_at=$5,
password_version=password_version+sqlc.arg(password_changed)::bigint
WHERE workspace_id=$1 AND id=$2 RETURNING *;

-- name: RevokeShareLink :one
UPDATE share_links SET revoked_at=coalesce(revoked_at,now()) WHERE workspace_id=$1 AND id=$2 RETURNING *;

-- name: RecordShareUse :one
UPDATE share_links SET last_used_at=now(),use_count=use_count+1
WHERE workspace_id=$1 AND id=$2 AND password_version=$3 AND revoked_at IS NULL
AND (expires_at IS NULL OR expires_at>now()) RETURNING *;
