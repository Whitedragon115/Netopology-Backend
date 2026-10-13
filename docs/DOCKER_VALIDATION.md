# Full-stack validation — 2026-10-14

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

WP-02's explicitly requested **local-engine** check remains unfinished:
Docker Desktop fails during inference-manager socket/listener startup.
No factory reset, data deletion or global Docker context change was attempted.
The successful remote test must not be reported as a successful local run.
