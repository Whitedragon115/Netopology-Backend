package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"go.yaml.in/yaml/v3"
)

// Compare every wire constraint recursively, independently of the Go registry.
// Reference names, numeric storage formats and prose do not change wire types.
func TestRecursiveContract(t *testing.T) {
	contract := loadYAML(t, "../../api/contract.yaml")
	raw, err := OpenAPI().YAML()
	if err != nil {
		t.Fatal(err)
	}
	var generated map[string]any
	if err := yaml.Unmarshal(raw, &generated); err != nil {
		t.Fatal(err)
	}
	for key, cop := range operations(contract) {
		gop, ok := operations(generated)[key]
		if !ok {
			t.Errorf("missing operation %s", key)
			continue
		}
		t.Run(str(cop["operationId"]), func(t *testing.T) {
			compareContent(t, contract, generated, asMap(dig(cop, "requestBody", "content")), asMap(dig(gop, "requestBody", "content")))
			for code, response := range asMap(cop["responses"]) {
				c := resolve(contract, asMap(response))
				g := resolve(generated, asMap(asMap(gop["responses"])[code]))
				compareContent(t, contract, generated, asMap(c["content"]), asMap(g["content"]))
			}
			for _, cp := range asSlice(cop["parameters"]) {
				cp := resolve(contract, asMap(cp))
				for _, gp := range asSlice(gop["parameters"]) {
					gp := resolve(generated, asMap(gp))
					if cp["name"] == gp["name"] && cp["in"] == gp["in"] {
						compareRecursiveSchema(t, str(cp["name"]), contract, generated, cp["schema"], gp["schema"])
					}
				}
			}
		})
	}
	// Also enforce public client aliases and the SSE data-line payload, which are
	// components rather than direct JSON response roots.
	for name, schema := range asMap(dig(contract, "components", "schemas")) {
		compareRecursiveSchema(t, name, contract, generated, schema, dig(generated, "components", "schemas", name))
	}
}

func compareContent(t *testing.T, cdoc, gdoc, c, g map[string]any) {
	t.Helper()
	for media, cm := range c {
		gm, ok := g[media]
		if !ok {
			t.Errorf("missing media type %s", media)
			continue
		}
		compareRecursiveSchema(t, media, cdoc, gdoc, asMap(cm)["schema"], asMap(gm)["schema"])
	}
}

func compareRecursiveSchema(t *testing.T, label string, cdoc, gdoc map[string]any, c, g any) {
	t.Helper()
	cn, gn := normalizeSchema(cdoc, c), normalizeSchema(gdoc, g)
	cb, _ := json.Marshal(cn)
	gb, _ := json.Marshal(gn)
	if string(cb) != string(gb) {
		t.Errorf("%s recursive schema differs\ncontract: %s\ngo:       %s", label, cb, gb)
	}
}

func normalizeSchema(doc map[string]any, value any) any {
	switch value := value.(type) {
	case []any:
		out := make([]any, len(value))
		for i, v := range value {
			out[i] = normalizeSchema(doc, v)
		}
		return out
	case map[string]any:
		out := map[string]any{}
		if ref := str(value["$ref"]); ref != "" {
			for k, v := range asMap(normalizeSchema(doc, resolve(doc, map[string]any{"$ref": ref}))) {
				out[k] = v
			}
		}
		for key, v := range value {
			if slices.Contains([]string{"$ref", "description", "title", "example", "examples"}, key) {
				continue
			}
			out[key] = normalizeSchema(doc, v)
		}
		if all := asSlice(out["allOf"]); len(all) == 1 {
			delete(out, "allOf")
			for k, v := range asMap(all[0]) {
				out[k] = v
			}
		}
		if one := asSlice(out["oneOf"]); len(one) == 2 {
			for i, branch := range one {
				if reflect.DeepEqual(branch, map[string]any{"type": "null"}) {
					nonnull := asMap(one[1-i])
					if typ, ok := nonnull["type"].(string); ok {
						delete(out, "oneOf")
						for k, v := range nonnull {
							out[k] = v
						}
						out["type"] = []any{typ, "null"}
					}
				}
			}
		}
		if format := str(out["format"]); slices.Contains([]string{"double", "float", "int32", "int64"}, format) {
			delete(out, "format")
		}
		if ap, ok := out["additionalProperties"]; ok && (ap == true || reflect.DeepEqual(ap, map[string]any{})) {
			delete(out, "additionalProperties")
		}
		for _, key := range []string{"required", "enum", "type", "oneOf", "anyOf", "allOf"} {
			if list, ok := out[key].([]any); ok {
				slices.SortFunc(list, func(a, b any) int {
					aj, _ := json.Marshal(a)
					bj, _ := json.Marshal(b)
					return strings.Compare(string(aj), string(bj))
				})
			}
		}
		return out
	default:
		return value
	}
}

// The gate must catch changes below the root, including nullability and unions.
func TestRecursiveSchemaComparisonDetectsDrift(t *testing.T) {
	doc := map[string]any{}
	base := map[string]any{"type": "object", "properties": map[string]any{"items": map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []any{"a", "b"}}}}}
	for _, constraint := range []string{"enum", "type", "default", "minimum", "minLength", "readOnly"} {
		t.Run(constraint, func(t *testing.T) {
			raw, _ := json.Marshal(base)
			var changed map[string]any
			if err := json.Unmarshal(raw, &changed); err != nil {
				t.Fatal(err)
			}
			dig(changed, "properties", "items", "items").(map[string]any)[constraint] = fmt.Sprint("changed-", constraint)
			if reflect.DeepEqual(normalizeSchema(doc, base), normalizeSchema(doc, changed)) {
				t.Fatal("nested change was ignored")
			}
		})
	}
}

func TestWireSchemaValidation(t *testing.T) {
	registry := OpenAPI().Components.Schemas
	id := "0192f3a0-7c1e-7000-8000-000000000001"
	tests := []struct {
		name, schema string
		value        any
		valid        bool
	}{
		{"shape variant", "CanvasOperation", map[string]any{"op": "shape.create", "kind": "rectangle"}, true},
		{"invalid shape enum", "CanvasOperation", map[string]any{"op": "shape.create", "kind": "rect"}, false},
		{"missing variant field", "CanvasOperation", map[string]any{"op": "place"}, false},
		{"nullable placement size", "CanvasOperation", map[string]any{"op": "update", "placementId": id, "width": nil}, true},
		{"invalid UUID", "CanvasOperation", map[string]any{"op": "shape.delete", "shapeId": "invalid"}, false},
		{"empty patch", "CanvasUpdate", map[string]any{}, false},
		{"viewport lower bound", "Viewport", map[string]any{"x": 0.0, "y": 0.0, "zoom": 0.0}, false},
		{"duplicate grants", "CanvasGrants", map[string]any{"canvasIds": []any{id, id}}, false},
		{"nullable field value", "Field", map[string]any{"key": "ip", "label": "IP", "kind": "text", "value": nil}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &huma.ValidateResult{}
			huma.Validate(registry, registry.SchemaFromRef("#/components/schemas/"+tt.schema), huma.NewPathBuffer(nil, 0), huma.ModeWriteToServer, tt.value, result)
			if (len(result.Errors) == 0) != tt.valid {
				t.Fatalf("valid=%v, errors=%v", tt.valid, result.Errors)
			}
		})
	}
}
