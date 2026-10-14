# Backend core B2: recovery and undo review

Date: 2026-10-14. Scope: the 15 direct children of DRA-34 at review start, all 72 canonical API operations, their service/transaction paths, database cascades, session handling and SSE recovery. Review only: findings below are **not fixed** by this document.

## Conclusion

The backend does **not** currently guarantee that every action can be undone. Transactions normally roll back failed writes before commit, but that is different from reversing a successful action. Hard deletes have no restore API. Some mutations can return an error after committing. A password-reset race, replay ordering gap and reparenting side effects also need fixes.

## Meaning of recovery

- **Rollback:** an error before commit leaves the mutation, audit and outbox uncommitted together.
- **Compensation:** a later authorized operation can approximate the old state if the caller saved it. IDs, audit history, security tokens and concurrent edits may differ.
- **Exact undo:** restore previous resource identities, dependents and positions without overwriting concurrent work. The existing contract does not provide this for every mutation.
- **Operational restore:** restoring a database backup is a separate controlled procedure. Git revert alone does not restore data or reverse migrations. DRA-98 already tracks backup/restore work outside B2.

Security revocations should remain irreversible: password changes, disabled users, logout, removed identities and revoked share links must not revive old sessions. Recovery uses new credentials or links with current authorization.

## Findings, in priority order

### R1 — P1: successful deletion cannot be exactly undone

Inventory, canvas and workspace deletes are hard deletes. Foreign-key cascades remove dependent links, notes, placements, grants and shares; workspace deletion also removes workspace audit records. Recreating a named resource allocates a new UUID and cannot restore the graph. Audit/outbox records are not before-images or a persistent undo journal.

Evidence: [inventory deletion](../internal/inventory/write_node.go), [canvas deletion](../internal/canvas/write.go), [workspace deletion](../internal/workspace/service.go), [foreign keys and reparent triggers](../db/migrations/00001_init.sql). The isolated `TestRecoveryReviewHardDeleteCascades` confirms that deleting a node removes its link and attached note; recreating the node generates another ID.

Contract question: define which content operations need exact undo, retention, ownership, version conflicts and dependent-resource restoration; then approve trash/history/restore semantics and contract changes. Do not invent endpoints under the current immutable-contract rule. The existing canvas batch can submit inverses for reversible edits, but does not solve permanent resource restoration. Permissions must be checked again when undo is applied, including permissions.md §2 rules 5 and 8 and §3.

### R2 — P1: a failed response may hide a committed mutation

Inventory create/update/health methods commit, then call another service read to construct the response. Canvas ApplyOperations commits, then calls GetView. A cancelled context, database read error or intervening access change can make the request report failure after the mutation, audit and event committed. A retry may duplicate creation; optimistic client rollback can diverge from stored state.

Evidence: [CreateNode/UpdateNode/SetNodeHealth](../internal/inventory/write_node.go#L100), corresponding area/link/note writers, and [ApplyOperations](../internal/canvas/operations.go#L133). `TestRecoveryReviewFailedResponseAfterCommit` cancels immediately after successful COMMIT and verifies an error response with the node, audit and outbox still persisted.

Suggested repair: construct the authorized response within the transaction, and avoid making optional post-commit view loading turn a committed result into a failed operation. Network loss around commit still requires reconciliation or approved idempotency/receipt semantics; do not claim transaction-local response construction eliminates every ambiguous outcome.

### R3 — P1: old-password login can create a session after password reset

Login checks the current password without locking the user row before inserting its session. A password-reset transaction can hold the user lock, change the password and delete sessions. A login that already checked the old password waits at session insertion, then continues after the reset commits and creates a usable session.

Evidence: [Login transaction](../internal/auth/service.go#L98), [account password reset](../internal/account/users.go), session INSERT foreign key. `TestRecoveryReviewPasswordResetRace` controls this ordering and verifies that the old-password login succeeds and its cookie accesses `/me` with 200 after reset.

Suggested repair: serialize the final password/status check and session creation against password reset/user disabling using a user-row lock. Review OAuth session creation for the same status/revocation ordering; the OAuth variant was inspected, not reproduced with a real provider. This affects permission table §3 account/session actions and their revocation guarantees.

### R4 — P1: SSE resume can omit an event committed after the cursor

Replay filters `id > Last-Event-ID`, but UUID allocation happens before transaction commit. A transaction with a smaller UUID can commit after an event with a larger UUID that the client has already received. The smaller event is absent on reconnect. Live-stream deduplication acknowledges this ordering problem, but replay still relies on UUID order.

Evidence: [ListWorkspaceEventsAfter](../db/queries/platform.sql#L17), [stream replay](../internal/events/service.go). `TestRecoveryReviewSSECommitOrder` commits the newer UUID first and the older UUID later; replay after the newer ID omits the later commit.

Suggested repair: define a replay/reconciliation strategy that handles commit order, duplicates, deletion and retention. A sequence allocated before commit has the same ordering problem. A fresh snapshot on reconnect can reconcile current canvas state, but is not a complete history guarantee. Preserve per-event access checks/redaction under permissions.md §§2–4.

### R5 — P2: reparenting changes positions on other canvases

Reparent triggers update placement parent IDs without converting x/y to the new parent's coordinate basis. The migration comment expects API coordinate conversion, but inventory reparent and canvas reparent paths do not preserve every affected canvas's absolute position. Undoing only the current canvas misses these side effects.

Evidence: [reparent triggers](../db/migrations/00001_init.sql#L418), [inventory updates](../internal/inventory/write_node.go), [canvas operation SQL](../db/queries/canvas.sql). `TestRecoveryReviewReparentCoordinates` uses two different parent positions and proves a node jumps on the second canvas after a successful inventory reparent.

Suggested repair: capture old absolute positions and convert all affected placements inside the same transaction. Preserve permission checks and capture all side effects for inverse operations. Do not edit the merged migration directly.

### R6 — P2 / contract question: placeOn writes to a locked canvas

Inventory creation's preparePlaceOn validates access but does not check or serialize against canvas locking. Node creation succeeds with a placement on a locked canvas. Permissions.md §2 rule 8 says a locked canvas is read-only for everyone, but inventory creation responses do not currently declare 423.

Evidence: [preparePlaceOn](../internal/inventory/write_helpers.go#L116). `TestRecoveryReviewLockedPlaceOn` locks a canvas and then creates a node there successfully (201).

Contract question: confirm lock scope for inventory placements and cross-canvas reparenting, and align affected operation responses before implementation. Once specified, serialize lock checks and writes so concurrent locks cannot be bypassed.

### R7 — P2: DRA-91 remains pending, deleted-item names are missing

Inventory and canvas mutation audit metadata omit the item's name. Once an item is deleted, Activity cannot reliably label it from current state. An item name improves the audit display; it is not enough to reconstruct the deleted object.

Evidence: [recordInventoryChange](../internal/inventory/write_helpers.go#L189), [canvasChange](../internal/canvas/write.go#L18). Capture the authorized safe name before deletion and add it to mutation audit metadata. Follow permissions.md §4 and the no-secrets-in-audit rule. No contract shape change is necessary.

### R8 — P3: DRA-90 remains pending, initial SSE body is missing

The SSE handler flushes headers without first writing a body comment. This is not the requested immediate `: ok\n\n` write and flush; proxy buffering can delay visible connection opening until an event or heartbeat.

Evidence: [stream handler](../internal/events/handler.go#L54). Write and flush the initial comment, handle write failure, then verify through Next and nginx. Source inspection confirms the missing write; this review did not restart the remote stack or repeat both proxy-path checks. No contract shape change is necessary.

## Verification and limits

Seven isolated characterization probes ran against the configured remote PostgreSQL 18 test service. Six reproduce recovery gaps; one verifies correct rollback when outbox insertion fails. Each uses disposable test databases, not application data. No Docker engine or dev server was started and the existing frontend was preserved.

Local reproducer: `.task/reviewproof/review_test.go`; output: `.task/reviewproof/output.log`. Run `go tool task test -- ./.task/reviewproof -run TestRecoveryReview -count=1 -v`. These tests intentionally assert observed counterexamples and are excluded from normal CI discovery. Their PASS means the counterexample was reproduced, not that the desired recovery behavior passes. Convert them into permanent expected-correct regression tests when repairing the findings.

`CONTRACT_STRICT=1 go tool task check` passed with zero lint issues. Normal test results reused Go's cache where valid; the seven review probes ran fresh, with no integration skips. Existing permission/contract suites are valuable but do not cover the counterexamples above. Live Google/Discord authentication, every browser checkbox, full backup restoration and load/exhaustive interleaving tests were not newly verified.

## All B2 subissues

Statuses below are the tracker snapshot at review start, not an assertion that every acceptance checkbox is complete.

| Issue | Scope | Tracker state | Recovery review |
| --- | --- | --- | --- |
| DRA-14 | WP-01 Platform, environment, reference feature (setup) | done | Platform Tx/audit/outbox rollback checked; setup has no exact undo; operational restore separate. |
| DRA-15 | WP-03 Password sign-in, logout, providers, getMe | done | Session lifetimes/touch reviewed; logout compensates login; R3 reset race. |
| DRA-16 | WP-04 Accounts: /me, /users, instance settings | in_review | Accounts/settings reviewed; password/session revocation intentionally not reversible; R3. |
| DRA-17 | WP-05 Google + Discord sign-in and linking | in_review | OAuth state, identity linking and session paths inspected; fresh authorization for recovery; live-provider verification not repeated. |
| DRA-18 | WP-06 Workspaces | in_review | Workspace CRUD reviewed; delete is destructive and cascades audit/content; R1. |
| DRA-19 | WP-07 Members and canvas grants | in_review | Owner/grant guards reviewed; removing members or roles needs captured grants for compensation; no resurrection of revoked access. |
| DRA-20 | WP-08 Inventory reads, visibility set, redaction | in_review | Visibility/redaction/read projections inspected; ordinary reads need no inverse; lazy area canvas is WP-10. |
| DRA-21 | WP-09 Inventory writes | in_review | Writes/cascades/placeOn/version checks reviewed; R1/R2/R5/R6/R7. |
| DRA-22 | WP-10 Canvases, canvas view, operations, lock | in_review | Batch transaction, operations, views, locks and side effects reviewed; R1/R2/R5/R6/R7. |
| DRA-23 | WP-11 Live events (SSE) | done | Broker/replay/projection/auth rechecks inspected; late-commit replay probe exposes R4; R8 pending. |
| DRA-24 | WP-12 Share links and guests | in_review | Guest projection, cookie/password version, expiry/revocation/accounting inspected; revoke uses a fresh link for recovery. |
| DRA-25 | WP-13 Audit log listing | in_review | Read-only listing, filters and keyset pagination inspected; audit is not undo storage; R7. |
| DRA-26 | WP-14 Permission matrix sweep, strict contract | in_review | Strict registration/schema/permission gates pass; current gates do not prove undo or these race cases. |
| DRA-90 | SSE: send a comment line right after the stream opens | todo | Pending: R8; add comment and proxy verification. |
| DRA-91 | Audit log: keep the item name in metadata | todo | Pending: R7; preserve item name before deletion. |

## Every canonical operation

This matrix states available recovery semantics, not a new undo guarantee. Ordinary authenticated reads can touch session expiry; middleware activity is not resource undo. Inverse writes must use current permissions, applicable If-Match versions and saved previous values. Creation/deletion compensation retains audit history and may allocate different IDs.

| WP | Operation | Route | Recovery assessment |
| --- | --- | --- | --- |
| WP-01 | `getSetupStatus` | `GET /setup` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-01 | `completeSetup` | `POST /setup` | One-time initialization; no reset endpoint; restore requires controlled backup procedure (R1). |
| WP-03 | `login` | `POST /auth/login` | Logout compensates session creation, not audit/last-login; password-reset race (R3). |
| WP-03 | `listAuthProviders` | `GET /auth/providers` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-05 | `startOAuthSignIn` | `GET /auth/oauth/{provider}/start` | Security flow; unlink/logout/re-authentication are compensation, not restoration of tokens. |
| WP-05 | `finishOAuthSignIn` | `GET /auth/oauth/{provider}/callback` | Security flow; unlink/logout/re-authentication are compensation, not restoration of tokens. |
| WP-03 | `logout` | `POST /auth/logout` | Security revocation is intentional; use fresh credentials/identity/link, never resurrect old sessions (R1/R3). |
| WP-03 | `getMe` | `GET /me` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-04 | `updateMe` | `PATCH /me` | Conditional old-value compensation only; capture prior state, reauthorize and reconcile versions. Security effects may be irreversible. |
| WP-04 | `changeMyPassword` | `POST /me/password` | Security revocation is intentional; use fresh credentials/identity/link, never resurrect old sessions (R1/R3). |
| WP-05 | `listMyIdentities` | `GET /me/identities` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-05 | `linkMyIdentity` | `GET /me/identities/{provider}/link` | Security flow; unlink/logout/re-authentication are compensation, not restoration of tokens. |
| WP-05 | `unlinkMyIdentity` | `DELETE /me/identities/{identityId}` | Security revocation is intentional; use fresh credentials/identity/link, never resurrect old sessions (R1/R3). |
| WP-04 | `listUsers` | `GET /users` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-04 | `createUser` | `POST /users` | Disable is compensation; no delete-user endpoint or exact reversal (R1). |
| WP-04 | `lookupUser` | `GET /users/lookup` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-04 | `getUser` | `GET /users/{userId}` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-04 | `updateUser` | `PATCH /users/{userId}` | Conditional old-value compensation only; capture prior state, reauthorize and reconcile versions. Security effects may be irreversible. |
| WP-04 | `resetUserPassword` | `POST /users/{userId}/password` | Security revocation is intentional; use fresh credentials/identity/link, never resurrect old sessions (R1/R3). |
| WP-04 | `getInstanceSettings` | `GET /instance/settings` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-04 | `updateInstanceSettings` | `PATCH /instance/settings` | Conditional old-value compensation only; capture prior state, reauthorize and reconcile versions. Security effects may be irreversible. |
| WP-06 | `listWorkspaces` | `GET /workspaces` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-06 | `createWorkspace` | `POST /workspaces` | Delete/revoke can compensate creation but audit, IDs and side effects remain; response uncertainty R2 for inventory. |
| WP-06 | `getWorkspace` | `GET /workspaces/{wsId}` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-06 | `updateWorkspace` | `PATCH /workspaces/{wsId}` | Conditional old-value compensation only; capture prior state, reauthorize and reconcile versions. Security effects may be irreversible. |
| WP-06 | `deleteWorkspace` | `DELETE /workspaces/{wsId}` | Hard deletion/removal; no exact restore endpoint. Cascades/grants need prior state and stable IDs (R1). |
| WP-07 | `listMembers` | `GET /workspaces/{wsId}/members` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-07 | `addMember` | `POST /workspaces/{wsId}/members` | Remove compensates if still permitted; audit and concurrent changes remain. |
| WP-07 | `updateMember` | `PATCH /workspaces/{wsId}/members/{userId}` | Conditional old-value compensation only; capture prior state, reauthorize and reconcile versions. Security effects may be irreversible. |
| WP-07 | `removeMember` | `DELETE /workspaces/{wsId}/members/{userId}` | Hard deletion/removal; no exact restore endpoint. Cascades/grants need prior state and stable IDs (R1). |
| WP-07 | `setMemberCanvasGrants` | `PUT /workspaces/{wsId}/members/{userId}/canvas-grants` | Restore captured previous grants only after fresh authorization and concurrency reconciliation. |
| WP-08 | `listAreas` | `GET /workspaces/{wsId}/areas` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-09 | `createArea` | `POST /workspaces/{wsId}/areas` | Delete/revoke can compensate creation but audit, IDs and side effects remain; response uncertainty R2 for inventory. |
| WP-08 | `getArea` | `GET /workspaces/{wsId}/areas/{areaId}` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-09 | `updateArea` | `PATCH /workspaces/{wsId}/areas/{areaId}` | Old values permit conditional compensation, but reparenting changes other canvases (R5); post-commit response R2. |
| WP-09 | `deleteArea` | `DELETE /workspaces/{wsId}/areas/{areaId}` | Hard deletion/removal; no exact restore endpoint. Cascades/grants need prior state and stable IDs (R1). |
| WP-10 | `getAreaCanvasView` | `GET /workspaces/{wsId}/areas/{areaId}/canvas` | Lazy creation is transactional; no exact reverse endpoint; R1. |
| WP-08 | `listNodes` | `GET /workspaces/{wsId}/nodes` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-09 | `createNode` | `POST /workspaces/{wsId}/nodes` | Delete/revoke can compensate creation but audit, IDs and side effects remain; response uncertainty R2 for inventory. |
| WP-08 | `getNode` | `GET /workspaces/{wsId}/nodes/{nodeId}` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-09 | `updateNode` | `PATCH /workspaces/{wsId}/nodes/{nodeId}` | Old values permit conditional compensation, but reparenting changes other canvases (R5); post-commit response R2. |
| WP-09 | `deleteNode` | `DELETE /workspaces/{wsId}/nodes/{nodeId}` | Hard deletion/removal; no exact restore endpoint. Cascades/grants need prior state and stable IDs (R1). |
| WP-08 | `getNodeUsage` | `GET /workspaces/{wsId}/nodes/{nodeId}/usage` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-09 | `setNodeHealth` | `PUT /workspaces/{wsId}/nodes/{nodeId}/health` | Conditional old-value write with current authorization/version; post-commit response R2. |
| WP-08 | `listLinks` | `GET /workspaces/{wsId}/links` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-09 | `createLink` | `POST /workspaces/{wsId}/links` | Delete/revoke can compensate creation but audit, IDs and side effects remain; response uncertainty R2 for inventory. |
| WP-08 | `getLink` | `GET /workspaces/{wsId}/links/{linkId}` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-09 | `updateLink` | `PATCH /workspaces/{wsId}/links/{linkId}` | Conditional old-value write with current authorization/version; post-commit response R2. |
| WP-09 | `deleteLink` | `DELETE /workspaces/{wsId}/links/{linkId}` | Hard deletion/removal; no exact restore endpoint. Cascades/grants need prior state and stable IDs (R1). |
| WP-08 | `listNotes` | `GET /workspaces/{wsId}/notes` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-09 | `createNote` | `POST /workspaces/{wsId}/notes` | Delete/revoke can compensate creation but audit, IDs and side effects remain; response uncertainty R2 for inventory. |
| WP-08 | `getNote` | `GET /workspaces/{wsId}/notes/{noteId}` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-09 | `updateNote` | `PATCH /workspaces/{wsId}/notes/{noteId}` | Conditional old-value write with current authorization/version; post-commit response R2. |
| WP-09 | `deleteNote` | `DELETE /workspaces/{wsId}/notes/{noteId}` | Hard deletion/removal; no exact restore endpoint. Cascades/grants need prior state and stable IDs (R1). |
| WP-10 | `listCanvases` | `GET /workspaces/{wsId}/canvases` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-10 | `createCanvas` | `POST /workspaces/{wsId}/canvases` | Delete/revoke can compensate creation but audit, IDs and side effects remain; response uncertainty R2 for inventory. |
| WP-10 | `getCanvas` | `GET /workspaces/{wsId}/canvases/{canvasId}` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-10 | `updateCanvas` | `PATCH /workspaces/{wsId}/canvases/{canvasId}` | Conditional old-value compensation only; capture prior state, reauthorize and reconcile versions. Security effects may be irreversible. |
| WP-10 | `deleteCanvas` | `DELETE /workspaces/{wsId}/canvases/{canvasId}` | Hard deletion/removal; no exact restore endpoint. Cascades/grants need prior state and stable IDs (R1). |
| WP-10 | `getCanvasView` | `GET /workspaces/{wsId}/canvases/{canvasId}/view` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-10 | `applyCanvasOperations` | `POST /workspaces/{wsId}/canvases/{canvasId}/operations` | Batch rollback before commit; inverses require saved prior state. Removed identities/cascades not universally restorable; R1/R2/R5. |
| WP-10 | `lockCanvas` | `POST /workspaces/{wsId}/canvases/{canvasId}/lock` | Opposite operation conditionally compensates; lock owner/timestamp/audit are not restored. |
| WP-10 | `unlockCanvas` | `DELETE /workspaces/{wsId}/canvases/{canvasId}/lock` | Opposite operation conditionally compensates; lock owner/timestamp/audit are not restored. |
| WP-12 | `listShareLinks` | `GET /workspaces/{wsId}/canvases/{canvasId}/share-links` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-12 | `createShareLink` | `POST /workspaces/{wsId}/canvases/{canvasId}/share-links` | Delete/revoke can compensate creation but audit, IDs and side effects remain; response uncertainty R2 for inventory. |
| WP-12 | `updateShareLink` | `PATCH /workspaces/{wsId}/share-links/{shareLinkId}` | Conditional old-value compensation only; capture prior state, reauthorize and reconcile versions. Security effects may be irreversible. |
| WP-12 | `revokeShareLink` | `DELETE /workspaces/{wsId}/share-links/{shareLinkId}` | Security revocation is intentional; use fresh credentials/identity/link, never resurrect old sessions (R1/R3). |
| WP-12 | `getShareInfo` | `GET /share/{token}` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-12 | `unlockShare` | `POST /share/{token}/unlock` | Session and usage accounting change; cookie expiry/sign-out is compensation; usage/audit remain. |
| WP-12 | `getSharedCanvasView` | `GET /share/{token}/view` | No user-data inverse needed; authentication middleware may touch session expiry. |
| WP-11 | `streamEvents` | `GET /workspaces/{wsId}/events` | Read stream; replay can miss late commits (R4); opening flush pending (R8). |
| WP-13 | `listAuditLog` | `GET /workspaces/{wsId}/audit-log` | No user-data inverse needed; authentication middleware may touch session expiry. |

## Function coverage boundary

[B2_FUNCTION_INDEX.md](B2_FUNCTION_INDEX.md) inventories every non-generated production Go function declaration under internal/. Service entry points and mutation helpers are mapped through the operation matrix and findings; transaction, auth, projection, crypto/config and stream infrastructure were inspected. Handler and conversion declarations are indexed for traceability. This is not a claim that all helper branches or every concurrency ordering have individual test coverage. Generated dbgen code and tests are excluded from that index; SQL sources and schema triggers were reviewed separately.

## Repair order

1. Close the password-reset/session race (R3), add a deterministic expected-correct concurrency test.
2. Eliminate avoidable post-commit error responses (R2); specify retry/reconciliation expectations.
3. Fix SSE reconnect reconciliation (R4), then the initial stream write already tracked by DRA-90.
4. Preserve cross-canvas positions on reparent (R5); resolve the lock contract question (R6).
5. Complete DRA-91 with safe item names and deletion audit assertions.
6. Approve precise content undo/restore design (R1) and implement it with concurrency, permissions and stable-ID restoration tests. Keep credential/session revocation outside exact undo. Coordinate controlled backup/restore with existing DRA-98.

## Tracker follow-up

The project issue inventory was read completely (104 issues across two pages) before creating findings. Existing DRA-90, DRA-91 and DRA-98 were reused for their scopes. Six additional B2 children now track the review gaps:

| Finding | Tracker issue |
| --- | --- |
| R1: exact content undo/restore design | DRA-117 |
| R3: password-reset/session race | DRA-118 |
| R2: errors after committed mutations | DRA-119 |
| R4: replay across out-of-order commits | DRA-120 |
| R5: cross-canvas reparent coordinates | DRA-121 |
| R6: inventory placement and lock contract | DRA-122 |

B2 remains in review. The 15-child table above describes the original review scope; these six findings expand its current child count to 21. Tracker updates contain high-level behavior and acceptance criteria, with no infrastructure addresses, secrets or commit IDs.
