-- Feature: setup (internal/setup). Reference implementation, see docs/ARCHITECTURE.md §12.

-- name: CountUsers :one
SELECT count(*) FROM users;

-- name: TryLockSetup :one
-- Serializes concurrent completeSetup calls inside one transaction (released at commit).
SELECT pg_try_advisory_xact_lock(hashtext('netopology.setup'));
