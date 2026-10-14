# Generated OpenAPI equivalence (DRA-72)

The generated `api/openapi.gen.yaml` now matches the canonical contract's wire
constraints recursively. `pnpm gen:api` uses this generated document. The
hand-written `api/contract.yaml` remains authoritative and was not modified.

`TestRecursiveContract` checks every operation's request, response and parameter
schemas, every public component, and the SSE Event payload. It checks nested
properties, required fields, nullability, enum/const values, defaults, bounds,
read/write flags, uniqueness and discriminated unions. The original contract
test also checks coverage, permission keys, security and status codes.

The comparator resolves references and equivalent single-allOf / nullable
oneOf representations. It ignores descriptive prose/examples, component
reference spelling and numeric Go storage formats (`int32`, `int64`, `double`).
Omitted additionalProperties and true/empty schemas are equivalent under JSON
Schema. These normalizations do not discard wire constraints.

CanvasOperation now publishes and validates all eight discriminated variants.
Go reflection is supplemented with explicit wire metadata for constraints it
cannot infer. Public client aliases derive from actual Go wire schemas;
x-client-schemas retains aliases and x-sse-event-schema retains Event through
Huma's unused-component pruning.

Tests assert that nested drift is detected and validate representative union,
nullability, enum, UUID, bounds, empty-patch and uniqueness constraints.
The frontend generator preserves its existing default-non-nullable option;
frontend type compatibility is checked separately.
