# Permission matrix coverage (WP-14)

References are to `../docs/design/permissions.md` §5. Tests use the real HTTP
stack and a PostgreSQL 18 database with RLS enabled.

| Row | Tests |
| --- | --- |
| 1: non-members get 404 | server/TestWorkspacePermissionMatrix; inventory/TestInventoryReadPermissionMatrix; events/TestStreamVisibilityRedactionAndResume; share/TestSharePermissionsExpiryAndRateLimit |
| 2: viewer writes denied; reads redacted | server/TestWorkspacePermissionMatrix covers every workspace mutation category except permitted self-leave; inventory read/redact tests; events stream tests |
| 3: editor grants, placeOn and external usage | inventory write/read tests; canvas operations tests |
| 4: admin protection and owner role restrictions | server/TestWorkspacePermissionMatrix; workspace/TestWorkspaceAdminCannotUnprotect; inventory write tests; member tests |
| 5: last owner | member tests |
| 6: guest isolation, expiry/revoke, password/rate limit | share/TestGuestViewAndLinkLifecycle; share/TestSharePermissionsExpiryAndRateLimit; inventory guest redaction unit tests |
| 7: instance/workspace role separation | account/TestAccountPermissionMatrix; server/TestWorkspacePermissionMatrix; workspace lifecycle tests |
| 8: setup only once | setup/TestFirstRunSetup |
| 9: locked canvas | server/TestWorkspacePermissionMatrix covers owner/admin/editor; canvas operations tests |
| 10: stale ETag | inventory write tests; workspace lifecycle tests; canvas operations tests |
| 11: provider identity rules | auth OAuth tests; account identity/password tests |

CI sets `CONTRACT_STRICT=1`, so missing contract operations now fail the build.
See CONTRACT_DIFFERENCES.md for the remaining generated-schema follow-up.
