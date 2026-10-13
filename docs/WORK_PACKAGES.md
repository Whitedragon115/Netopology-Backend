# Work Packages (phase B2 → M2)

Each package is one pull request, sized for one agent. Read [AGENTS.md](../AGENTS.md) first.
Every package ends with `go tool task check` green and the frontend contract unchanged.

## Status

| WP | Title | Depends on | State |
|---|---|---|---|
| 01 | Platform, environment, reference feature (setup) | — | ✅ done (architect) |
| 02 | Full stack in Docker (parent repo compose) | 01 | 🟡 files done, both images build; first `docker compose up` on a local engine still to verify |
| 03 | Password sign-in, logout, providers, `getMe` | 01 | ✅ done |
| 04 | Accounts: `/me`, `/users`, instance settings | 03 | ✅ done |
| 05 | Google + Discord sign-in and linking | 04 | ✅ done |
| 06 | Workspaces | 03 | ✅ done |
| 07 | Members and canvas grants | 06 | ✅ done |
| 08 | Inventory reads, visibility set, redaction | 07 | ⏳ |
| 09 | Inventory writes | 08 | ⏳ |
| 10 | Canvases, canvas view, operations, lock | 09 | ⏳ |
| 11 | Live events (SSE) | 08 (10 for canvas events) | ⏳ |
| 12 | Share links and guests | 10 | ⏳ |
| 13 | Audit log listing | 06 | ⏳ |
| 14 | Permission matrix sweep, strict contract | 03–13 | ⏳ |
| 15 | Frontend on the real API (parent repo) | 10 (12 for sharing) | ⏳ |

Parallel lanes after WP-03: **A** 04 → 05 · **B** 06 → 07 → 08 → 09 → 10 → 12 · **C** 13 (after 06) · **D** 11 (after 08).
Update the State column in your PR.

---

## WP-01 Platform, environment, reference feature ✅

Delivered: everything in [ARCHITECTURE.md §2](ARCHITECTURE.md#2-repository-layout) except the feature
bodies; `getSetupStatus` + `completeSetup` implemented and tested; contract, permission-key and
event-type tests; test harness; Taskfile; lint; CI; Dockerfile; migration `00003_events.sql`.

## WP-02 Full stack in Docker 🟡 (verify)

Parent repo: `compose.yaml`, `Dockerfile` (web), `deploy/nginx/default.conf`, `.env.example`.
**Verify:** on a machine with a local Docker engine, `cp .env.example .env`, fill the secrets,
`docker compose up --build` → http://localhost shows the app; `/api/v1/setup` returns JSON.
Fix anything that does not start; do not change the network layout (IDEA.md §14.2).

---

## WP-03 Password sign-in, logout, providers, `getMe`

**Package:** `internal/auth` (login, logout, providers), `internal/account` (`getMe` only).
**Operations:** `login`, `logout`, `listAuthProviders`, `getMe`.
**Queries:** `users.sql` (exists), `sessions.sql` (exists), add `ListIdentityProviders` usage.

Rules (ARCHITECTURE §6.2): constant-time failure path (`DummyVerify`), one 401 code
`auth.invalid_credentials` for unknown user / wrong password / disabled; rate limit per IP and per
username (`internal/auth/ratelimit.go`, token bucket, injectable clock for tests) → 429 + `Retry-After`;
`remember` per the contract's `LoginRequest.remember` description (ARCHITECTURE §6.1); rehash old hashes; audit
success and failure. `logout` deletes the session row, clears the cookie, 204. `listAuthProviders`:
`password: true`, providers from config (`GoogleEnabled`/`DiscordEnabled`). `getMe` returns
`apimodel.Account` of the principal (401 if anonymous).

Also: implement session lifetimes exactly as ARCHITECTURE §6.1 "Lifetimes" (migration
`00004_session_remember.sql`, `auth.RememberTTL`, the touch query using the session's own lifetime),
and add `testutil.(*API).Login(username, password) *Client`.

**Done when:** tests for each rule above, including "rate limit returns 429 with Retry-After" and
"session cookie stops working after logout".

## WP-04 Accounts

**Package:** `internal/account`. **Operations:** `updateMe`, `changeMyPassword`, `listMyIdentities`,
`unlinkMyIdentity`, `listUsers`, `createUser`, `lookupUser`, `getUser`, `updateUser`,
`resetUserPassword`, `getInstanceSettings`, `updateInstanceSettings`.
**Queries:** finish `users.sql` (TODO list in the file), add `instance_settings.sql`.

Rules (permissions.md §3 "Instance", rules 9, 10):
- `changeMyPassword` requires the current password when the account has one; setting a first password
  on a provider-only account requires a provider sign-in within the last 10 minutes (store
  `sessions.created_at` + the identity's `last_used_at` check).
- `unlinkMyIdentity` / password removal must keep ≥ 1 sign-in method → 409 `auth.last_sign_in_method`.
- Only SO creates server admins; SA cannot modify SO or other SAs, cannot grant `admin`/`owner`;
  SO can transfer ownership (old owner becomes `admin`, in one transaction respecting
  `users_single_instance_owner`).
- `resetUserPassword` and disabling a user end all their sessions.
- `lookupUser`: caller must be owner/admin of at least one workspace; exact username match; returns
  only `id, username, displayName` (contract `User`).
- Instance settings: keys `sessionTtlHours`, `oauthAutoLinkByEmail` in `instance_settings` (one row per
  key, jsonb value), defaults from the contract when absent.

Add `testutil` helper `CreateUser(username, role)` returning a signed-in client.

## WP-05 Google and Discord

**Package:** `internal/auth` (`oauth.go`), `internal/account` (`linkMyIdentity`).
**Operations:** `startOAuthSignIn`, `finishOAuthSignIn`, `linkMyIdentity`.
Follow ARCHITECTURE §6.3 exactly. Provider endpoints behind a small interface so tests use an
`httptest` fake provider (no real Google/Discord in tests). Test: linked login, auto-link by verified
email, unverified email refused, unlinked refused (no account created), link mode, identity in use,
tampered/expired state cookie, `returnTo` that is not a same-origin path is ignored.

## WP-06 Workspaces

**Package:** `internal/workspace`. **Operations:** `listWorkspaces`, `createWorkspace`,
`getWorkspace`, `updateWorkspace`, `deleteWorkspace`. **Queries:** `workspaces.sql` TODOs.

Rules: list = only memberships, each with the caller's `membership`; create = server owner/admin only,
creator becomes owner (`InsertMember`, `can_view_sensitive=true`), slug from `slug.Make` when not
given, `slug.taken` on conflict; update honors `If-Match`, admin may set but not clear
`isProtected` (`403 protected`); delete = owner only, not while protected. ETag on reads/updates.
Events: none in v1 (workspace changes are rare; the frontend refetches on focus).

## WP-07 Members and canvas grants

**Package:** `internal/member`. **Operations:** `listMembers`, `addMember`, `updateMember`,
`removeMember`, `setMemberCanvasGrants`.

Rules 7 and 9 of permissions.md, plus: editors/viewers see names and roles only (no
`canViewSensitive`, no `addedBy`) per the contract; `removeMember` by E/V only for themselves
(leave); last owner → 409 `member.last_owner` (lock owner rows `FOR UPDATE`); grants only for
members with role `editor` and only tab canvases (`area_id IS NULL`); replacing grants is one
statement set inside `InWorkspace`. Emit `member.changed`. Add `testutil` `AddMember(ws, user, role)`
and `ClientAs(role)` (creates a user, adds them, returns a signed-in client).

## WP-08 Inventory reads, visibility, redaction

**Package:** `internal/inventory`. **Operations:** `listAreas`, `getArea`, `listNodes`, `getNode`,
`getNodeUsage`, `listLinks`, `getLink`, `listNotes`, `getNote`.

The heart of authorization. Deliver:
1. `db/queries/inventory.sql` with the **visibility CTEs** (ARCHITECTURE §7) and list/get queries that
   take an optional "editor user id" parameter (NULL = manager/viewer = no filter).
2. Wire types matching the contract (`Node`, `Field`, `Health`, `Area`, `Link`, `Note`, ...).
3. Redaction functions (ARCHITECTURE §8) with table-driven unit tests for every row of permissions.md §4.
4. An exported read interface other features use, e.g.
   `func (s *Service) ReadForActor(ctx, q, actor, ids) (Snapshot, error)` and a guest variant
   (decide the exact shape; document it in the package comment).
5. Filters from the contract (e.g. `listNodes` by area/category/canvas) and cursor pagination.
6. `getNodeUsage`: canvases using the node; editors see inaccessible canvases only as a count.

## WP-09 Inventory writes

**Operations:** `createArea`, `updateArea`, `deleteArea`, `createNode`, `updateNode`, `deleteNode`,
`setNodeHealth`, `createLink`, `updateLink`, `deleteLink`, `createNote`, `updateNote`, `deleteNote`.

Rules 4, 5, 6 (ARCHITECTURE §7), `If-Match`, ETags, audit, events with the canvases showing the
item. `placeOn` creates the placement in the same transaction (reuse the placement insert from
WP-10's queries or write it here and let WP-10 reuse it). Node slug generated from the name when
omitted (`slug.Make` + numeric suffix on conflict). Validate `properties` (array of `Field`) and area
`cidr`. Translate trigger errors (area cycles) to 409/422 with a clear code. Links: editor needs both
endpoints visible. Workspace-level notes: owner/admin only. `setNodeHealth` (manual) emits `health.changed`.

## WP-10 Canvases

**Package:** `internal/canvas`. **Operations:** `listCanvases`, `createCanvas`, `getCanvas`,
`updateCanvas`, `deleteCanvas`, `getCanvasView`, `getAreaCanvasView`, `applyCanvasOperations`,
`lockCanvas`, `unlockCanvas`.

- **View** (`CanvasView`): canvas + placements + the inventory items they reference (redacted, via the
  inventory read interface) + visible links from `canvas_visible_links` + notes + shapes + viewer info.
  For area canvases include the boundary-crossing links and ghost endpoints exactly as the contract
  describes; the frontend mock `mocks/handlers.ts → buildView` is a working reference.
- `getAreaCanvasView` creates the inner canvas on first open (rule 11), race-safe
  (`INSERT ... ON CONFLICT (area_id) DO NOTHING` then select).
- **Operations**: the contract's operation list applied **atomically** (all or nothing) in one
  `InWorkspace` transaction; editors only on granted canvases (and area canvases per rule 11);
  locked → 423; trigger errors → 409 with the failing operation index in `detail`; one audit entry
  (`canvas.edit`, metadata = operation kinds and counts) and one `canvas.operations_applied` event.
- Lock/unlock set `locked_by/locked_at`; emit `canvas.updated`.
- Editors cannot create/delete canvases (role table), viewers see all tab canvases, editors only granted.

## WP-11 Live events

**Packages:** `internal/platform/events` (`Broker.Run`), `internal/events` (`streamEvents`).
Follow ARCHITECTURE §9. Use `huma/v2/sse` or a raw handler registered through `httpx.Op` (the
operation must still appear in the generated spec with the contract's shape). Tests: event arrives
after commit and not after rollback; viewer receives redacted data; editor does not receive events
for invisible items; `?canvasId=` filter; resume with `Last-Event-ID`; non-member → 404.

## WP-12 Share links and guests

**Package:** `internal/share`. **Operations:** `listShareLinks`, `createShareLink`,
`updateShareLink`, `revokeShareLink`, `getShareInfo`, `unlockShare`, `getSharedCanvasView`.

Token from `crypto.NewToken`, stored hashed, returned **once** on create (contract). Optional
password (argon2id). Expiry, revoke, `last_used_at`/`use_count`. Unlock rate-limited per link and IP.
Guest view = canvas view with **guest redaction** (permissions.md §4 "Guest" column: omit sensitive
fields and notes, health messages, user ids, unplaced items). Expired/revoked/unknown → 404; wrong
password → 401. Every non-share endpoint with only a guest cookie → 401.

## WP-13 Audit log listing

**Package:** `internal/audit`. **Operation:** `listAuditLog`. Owner/admin, cursor pagination
(newest first: keyset on `(created_at, id)` descending), filters from the contract. Add
`db/queries/audit.sql`. Actor display names joined from `users` (null for deleted users).

## WP-14 Permission matrix sweep

Make every row of permissions.md §5 an automated test (they may already exist per feature: collect
the missing ones). Switch CI to `CONTRACT_STRICT=1`. Point the frontend's `pnpm gen:api` at
`backend/api/openapi.gen.yaml` if it is now equivalent to the contract, or document the remaining
differences in an issue.

## WP-15 Frontend on the real API (parent repo)

Run the frontend with `NEXT_PUBLIC_API_MOCKING=disabled`. Fix every difference between the mock and
the real API **on the side that disagrees with the contract**. Add the sign-in and first-run screens
(the backend now supports them). Keep MSW for UI development, but it must stay contract-faithful.
