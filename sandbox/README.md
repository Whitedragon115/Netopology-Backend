# sandbox (phase C3, not started)

The container that runs plugins (JS/TS and Python) and user workflow scripts. Nothing here yet.

Design constraints already decided (IDEA.md §5.1, §14.2, §14.3):

- One container for all plugins: a Node host and a Python host, each a JSON-RPC 2.0 server over WebSocket.
- Only the `worker` talks to it, on the private `sandbox` network. The sandbox can NOT reach the
  database or the api; it can reach the internet and your LAN (it has to, to manage Cloudflare, NPM, routers...).
- Calls back into Netopology go to the worker through a narrow endpoint authenticated by a per-run token.
- Hardening: non-root, read-only root filesystem + tmpfs, `cap_drop: [ALL]`, `no-new-privileges`, CPU/memory/PID
  limits, per-run time limit enforced by the worker. Plugin packages mounted read-only. Never the Docker socket.
- Credentials are passed per call, kept in memory only, never written to disk or logs.

The worker side (client, run tokens, timeouts) will live in `internal/worker/sandbox/`.
