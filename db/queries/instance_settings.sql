-- Instance settings are not workspace scoped. Services authorize all access.

-- name: GetInstanceSetting :one
SELECT value FROM instance_settings WHERE key = $1;

-- name: UpsertInstanceSetting :exec
INSERT INTO instance_settings (key, value, updated_by)
VALUES ($1, $2, $3)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by;
