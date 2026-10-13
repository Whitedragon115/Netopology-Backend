package httpx

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
)

// Op builds the huma.Operation for one contract entry. Every handler registers through it, so
// each operation carries its x-permission and the right security requirement.
//
//	huma.Register(api, httpx.Op(authz.NodeRead, http.MethodGet, "/workspaces/{wsId}/nodes/{nodeId}",
//		"getNode", "Nodes", "Get a node", httpx.Errors(404)), h.get)
//
// The values must match api/contract.yaml exactly; `task test` runs the contract test.
func Op(perm authz.Permission, method, path, operationID, tag, summary string, opts ...OpOption) huma.Operation {
	op := huma.Operation{
		OperationID: operationID,
		Method:      method,
		Path:        path,
		Summary:     summary,
		Tags:        []string{tag},
		Extensions:  map[string]any{"x-permission": string(perm)},
	}
	switch authz.ScopeOf(perm) {
	case authz.ScopeAnonymous:
		op.Security = []map[string][]string{} // explicitly public
	case authz.ScopeGuest:
		op.Security = []map[string][]string{{"guest": {}}}
	default:
		// inherits the top-level `security: [session]`
	}
	if method == http.MethodPost && op.DefaultStatus == 0 {
		op.DefaultStatus = http.StatusOK
	}
	for _, o := range opts {
		o(&op)
	}
	return op
}

type OpOption func(*huma.Operation)

// Status sets the success status (201 for creates, 204 for deletes without body, ...).
func Status(code int) OpOption { return func(op *huma.Operation) { op.DefaultStatus = code } }

// Errors documents the error statuses this operation can return (see the contract's responses).
func Errors(codes ...int) OpOption {
	return func(op *huma.Operation) { op.Errors = append(op.Errors, codes...) }
}

// Describe sets the long description (Markdown), copied from the contract.
func Describe(text string) OpOption { return func(op *huma.Operation) { op.Description = text } }
