# Generated OpenAPI follow-up (WP-14)

The canonical frontend contract remains `api/contract.yaml`. Keep `pnpm gen:api`
pointed there until the generated document is proven equivalent. The strict CI
gate currently verifies operation coverage, permissions, parameters, top-level
request/response property names and required lists; it is not recursive schema
equivalence.

Known differences to resolve before switching the frontend generator:

- `CanvasOperation`: Go accepts one flat operation struct and validates each
  variant in the service; the canonical contract describes discriminated `oneOf`
  variants. Generated clients would lose per-variant required fields.
- Nested schemas, nullable fields, defaults, enum constraints and descriptions
  are not all recursively compared by the current contract test.
- SSE uses an explicit string response schema. Its payload is the contract
  Event; a follow-up should enforce every payload schema recursively too.

This is a generated-schema follow-up, not authorization to modify the canonical
contract. Runtime behavior and permission tests remain independent gates.
