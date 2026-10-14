package server

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

// Wire metadata that cannot be inferred from Go (a pointer can mean optional,
// nullable, or both). Keep this independent of contract.yaml: the recursive
// contract gate compares the resulting schemas against that specification.
type wireRegistry struct {
	huma.Registry
	patched map[string]bool
}

func newWireRegistry() *wireRegistry {
	return &wireRegistry{Registry: huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer), patched: map[string]bool{}}
}

func (r *wireRegistry) MarshalJSON() ([]byte, error) { return json.Marshal(r.Map()) }

func (r *wireRegistry) Schema(t reflect.Type, allowRef bool, hint string) *huma.Schema {
	s := r.Registry.Schema(t, allowRef, hint)
	for name, schema := range r.Map() {
		if r.patched[name] || schema.Type == "" && len(schema.Properties) == 0 {
			continue
		}
		r.patched[name] = true
		allowUnknownProperties(schema)
		if typ := r.TypeFromRef("#/components/schemas/" + name); typ != nil {
			wireUUIDs(schema, typ)
		}
		wireMetadata(name, schema)
		schema.PrecomputeMessages()
	}
	wireUUIDs(s, t)
	return s
}

func allowUnknownProperties(s *huma.Schema) {
	if s.Type == "object" && s.AdditionalProperties == false {
		s.AdditionalProperties = nil // omitted means allowed, as in the contract
	}
	for _, p := range s.Properties {
		allowUnknownProperties(p)
	}
	if s.Items != nil {
		allowUnknownProperties(s.Items)
	}
}

func wireUUIDs(s *huma.Schema, t reflect.Type) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == reflect.TypeFor[uuid.UUID]() {
		s.Format = "uuid" // Huma sees TextUnmarshaler and only infers string
		return
	}
	if t.Kind() == reflect.Slice && s.Items != nil {
		wireUUIDs(s.Items, t.Elem())
	}
	if t.Kind() == reflect.Struct {
		for field := range t.Fields() {
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if p := s.Properties[name]; p != nil && p.Ref == "" {
				wireUUIDs(p, field.Type)
			}
		}
	}
}

func nullable(s *huma.Schema) {
	if s.Ref != "" {
		ref := s.Ref
		*s = huma.Schema{OneOf: []*huma.Schema{{Ref: ref}, {Type: "null"}}}
	} else {
		s.Nullable = true
	}
}

func floatPtr(n float64) *float64 { return &n }
func intPtr(n int) *int           { return &n }

func wireMetadata(name string, s *huma.Schema) {
	p := s.Properties
	setNullable := func(fields ...string) {
		for _, field := range fields {
			if p[field] != nil {
				nullable(p[field])
			}
		}
	}
	setDefault := func(field string, value any) { p[field].Default = value }
	if slices.Contains([]string{"MeUpdate", "AccountUpdate", "WorkspaceUpdate", "MemberUpdate", "AreaUpdate", "NodeUpdate", "LinkUpdate", "NoteUpdate", "CanvasUpdate", "ShareLinkUpdate"}, name) {
		s.MinProperties = intPtr(1)
	}
	if version := p["version"]; version != nil {
		version.Minimum = floatPtr(1)
		version.ReadOnly = true
	}
	switch name {
	case "Field":
		setNullable("value")
		setDefault("sensitive", false)
		p["redacted"].ReadOnly = true
	case "Health":
		setNullable("checkedAt", "message")
	case "PlaceOn":
		p["width"].ExclusiveMinimum, p["height"].ExclusiveMinimum = floatPtr(0), floatPtr(0)
		setDefault("x", 0)
		setDefault("y", 0)
	case "Account", "PublicUser":
		p["username"].Pattern = "^[A-Za-z0-9][A-Za-z0-9._-]{1,38}[A-Za-z0-9]$"
	case "AccountCreate":
		s.AnyOf = []*huma.Schema{{Required: []string{"password"}}, {Required: []string{"email"}}}
	case "InstanceSettings":
		p["sessionTtlHours"].Nullable, p["oauthAutoLinkByEmail"].Nullable = false, false
	case "CanvasGrants":
		p["canvasIds"].UniqueItems = true
	case "Area":
		setNullable("cidr", "vlanId", "parentId")
		p["vlanId"].Minimum, p["vlanId"].Maximum = floatPtr(1), floatPtr(4094)
		p["health"].Enum = healthEnum()
		p["health"].ReadOnly, p["nodeCount"].ReadOnly = true, true
		setDefault("color", "yellow")
	case "AreaCreate":
		setDefault("color", "yellow")
	case "AreaUpdate":
		setNullable("cidr", "vlanId", "parentId")
		setDefault("color", "yellow")
	case "Node":
		setNullable("areaId", "createdBy", "icon")
	case "NodeCreate":
		p["slug"].Pattern = "^[a-z0-9][a-z0-9-]{0,62}$"
		setDefault("type", "builtin.generic")
		setDefault("isProtected", false)
	case "NodeUpdate":
		setNullable("areaId", "icon")
		p["slug"].Pattern = "^[a-z0-9][a-z0-9-]{0,62}$"
	case "AreaPage", "NodePage", "LinkPage", "NotePage":
		setNullable("nextCursor")
	case "Link", "LinkUpdate":
		setNullable("sourcePort", "targetPort")
	case "LinkCreate":
		setDefault("directed", false)
	case "Note":
		setNullable("createdBy", "updatedBy")
		p["redacted"].ReadOnly = true
	case "NoteCreate":
		setDefault("isSensitive", false)
	case "CanvasViewport":
		p["zoom"].ExclusiveMinimum, p["zoom"].Maximum = floatPtr(0), floatPtr(8)
	case "Canvas":
		setNullable("areaId", "lock")
		p["settings"] = canvasSettingsSchema()
	case "CanvasCreate", "CanvasUpdate":
		p["settings"] = canvasSettingsSchema()
		if name == "CanvasCreate" {
			setDefault("isProtected", false)
		}
	case "Placement", "Shape":
		setNullable("width", "height", "parentPlacementId")
		if name == "Shape" {
			p["kind"].Enum = shapeEnum()
		}
	case "LinkOverride":
		setNullable("waypoints")
	case "BreadcrumbItem":
		setNullable("areaId")
		s.Required = []string{"name", "canvasId"}
	case "Viewer":
		p["role"].Enum = []any{"owner", "admin", "editor", "viewer", "guest"}
	case "ShareLink":
		setNullable("createdBy")
	case "Actor":
		setNullable("id")
		s.Required = []string{"type"}
	case "AuditEntry":
		setNullable("target")
		s.Required = []string{"id", "actor", "action", "metadata", "createdAt"}
	case "Event":
		setNullable("canvasId", "actorId")
		p["data"] = &huma.Schema{Type: "object", AdditionalProperties: true}
	}
}

func healthEnum() []any {
	return []any{"healthy", "degraded", "down", "unknown"}
}

func shapeEnum() []any {
	return []any{"rectangle", "ellipse", "line", "arrow", "text", "freehand"}
}

func canvasSettingsSchema() *huma.Schema {
	return &huma.Schema{Type: "object", AdditionalProperties: true, Properties: map[string]*huma.Schema{
		"grid":            {Type: "boolean", Default: true},
		"snap":            {Type: "integer", Minimum: floatPtr(0), Default: 16},
		"showHealth":      {Type: "boolean", Default: true},
		"hiddenLinkKinds": {Type: "array", UniqueItems: true, Items: &huma.Schema{Type: "string", Enum: []any{"physical", "logical", "vpn_tunnel", "proxy_route", "dns_dependency", "storage_mount"}}},
	}}
}

// Keep the frontend's public type names even when Go uses inline scalar fields
// or different helper struct names. Derive each alias from actual wire types.
func publishClientSchemas(api huma.API) {
	doc := api.OpenAPI()
	schemas := doc.Components.Schemas.Map()
	aliases := map[string][2]string{
		"Id": {"Canvas", "id"}, "Slug": {"Workspace", "slug"}, "Timestamp": {"Canvas", "createdAt"},
		"Version": {"Canvas", "version"}, "Role": {"Membership", "role"}, "ViewerRole": {"Viewer", "role"},
		"HealthStatus": {"Health", "status"}, "NodeCategory": {"Node", "category"}, "LinkKind": {"Link", "kind"},
		"AreaKind": {"Area", "kind"}, "AreaColor": {"Area", "color"}, "NoteKind": {"Note", "kind"},
		"NoteColor": {"Note", "color"}, "ShapeKind": {"Shape", "kind"}, "Style": {"Placement", "style"},
		"AuthProvider": {"Identity", "provider"}, "InstanceRole": {"Account", "instanceRole"},
		"Username": {"User", "username"}, "Password": {"SetupRequest", "password"},
		"CanvasSettings": {"Canvas", "settings"},
	}
	refs := make([]*huma.Schema, 0, len(aliases)+2)
	for name, source := range aliases {
		copy := *schemas[source[0]].Properties[source[1]]
		schemas[name] = &copy
		refs = append(refs, &huma.Schema{Ref: "#/components/schemas/" + name})
	}
	for name, target := range map[string]string{"Me": "Account", "Viewport": "CanvasViewport"} {
		schemas[name] = &huma.Schema{Ref: "#/components/schemas/" + target}
		refs = append(refs, &huma.Schema{Ref: "#/components/schemas/" + name})
	}
	for _, schema := range schemas {
		if schema.Properties["version"] != nil {
			schema.Properties["version"] = &huma.Schema{Ref: "#/components/schemas/Version"}
		}
	}
	slices.SortFunc(refs, func(a, b *huma.Schema) int { return strings.Compare(a.Ref, b.Ref) })
	// Huma prunes unreferenced components. These aliases are also part of the
	// client contract, although routes already refer to their equivalent shapes.
	if doc.Extensions == nil {
		doc.Extensions = map[string]any{}
	}
	doc.Extensions["x-client-schemas"] = refs
}
