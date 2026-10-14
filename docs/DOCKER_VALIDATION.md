# Full-stack validation — 2026-10-14

## Final remote verification

The user explicitly selected the configured remote Docker context and instructed
that Docker Desktop must not be used. This overrides WP-02's original local
engine wording. The locally started Desktop instance was stopped.

The checked-in Compose configuration now builds nginx configuration and
PostgreSQL initialization into images, rather than mounting Windows paths on
the remote host. PostgreSQL readiness probes TCP, avoiding the temporary Unix
socket server used during first initialization. Edge/core network isolation and
the gateway routes are preserved.

An isolated remote Compose project with fresh credentials and its own volume
built all images and started all five services. Verified via its gateway:
frontend HTML, fresh first-run setup, authenticated session, workspace, canvas
and node persistence, a discriminated canvas operation, guest share unlock/view,
audit listing, SSE streaming and worker startup. PostgreSQL reported 18.6.

The existing remote PostgreSQL container and all frontend processes were
preserved. Temporary verification resources are removed after validation.

## Earlier verification and local recovery attempt

The parent Dockerfiles built successfully, including the Next.js production
build and the Go API/worker image. An isolated Compose project was then started
on the available remote Docker context and all five services were healthy.

Verified through its gateway on port 3100:

- `/setup` returned the frontend HTML page; `/api/v1/setup` returned JSON.
- First-run setup created a test owner/session and workspace in a fresh database.
- Canvas creation, node placement and canvas view readback succeeded.
- Share creation, guest unlock and guest canvas readback succeeded.
- Audit listing returned the mutations.
- SSE opened through nginx as `text/event-stream` without buffering.
- Worker startup succeeded; the test database reported PostgreSQL 18.6.

Remote Compose bind mounts cannot read this Windows checkout. For this test,
temporary images copied the existing nginx config and PostgreSQL init script
into their containers. The edge/core network layout and service behavior were
preserved. Temporary credentials and a separate database volume were used.

After verification, the isolated project and its verified test volume were
removed. The original database and other agents' containers were preserved.
No persistent application data was reset.

The earlier **local-engine** check initially remained unfinished:
Docker Desktop fails during inference-manager socket/listener startup.
No factory reset, data deletion or global Docker context change was attempted.
During the final retry, the stale runtime socket directories were quarantined
intact and Docker Desktop recovered. No application stack was started locally;
Desktop was stopped when the user clarified the remote-only requirement. No
local run is claimed. The quarantined runtime-only directories remain preserved.
