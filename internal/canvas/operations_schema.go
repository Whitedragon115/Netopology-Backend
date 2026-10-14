package canvas

import (
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// TransformSchema preserves the flat decoding struct while describing and
// validating the eight wire variants. Each const selects exactly one branch.
func (CanvasOperation) TransformSchema(r huma.Registry, flat *huma.Schema) *huma.Schema {
	union := &huma.Schema{Discriminator: &huma.Discriminator{PropertyName: "op", Mapping: map[string]string{}}}
	variants := []struct{ name, op, required, fields string }{
		{"OpPlace", "place", "item", "clientId item parentPlacementId x y width height zIndex collapsed style"},
		{"OpUpdatePlacement", "update", "placementId", "placementId parentPlacementId x y width height zIndex collapsed style"},
		{"OpRemovePlacement", "remove", "placementId", "placementId"},
		{"OpCreateShape", "shape.create", "kind", "clientId kind x y width height rotation zIndex style data"},
		{"OpUpdateShape", "shape.update", "shapeId", "shapeId x y width height rotation zIndex style data"},
		{"OpDeleteShape", "shape.delete", "shapeId", "shapeId"},
		{"OpLinkOverride", "link.override", "linkId", "linkId hidden style waypoints"},
		{"OpLinkReset", "link.reset", "linkId", "linkId"},
	}
	for _, variant := range variants {
		s := &huma.Schema{Type: "object", Required: []string{"op", variant.required}, Properties: map[string]*huma.Schema{"op": {Const: variant.op}}}
		for _, name := range strings.Fields(variant.fields) {
			property := *flat.Properties[name]
			if name == "placementId" || name == "shapeId" || name == "linkId" || name == "parentPlacementId" {
				property.Format = "uuid"
			}
			s.Properties[name] = &property
		}
		if variant.op == "place" {
			zero := 0.0
			s.Properties["width"].ExclusiveMinimum, s.Properties["height"].ExclusiveMinimum = &zero, &zero
			s.Properties["x"].Default, s.Properties["y"].Default = 0, 0
			s.Properties["collapsed"].Default = false
		}
		if variant.op == "shape.create" {
			s.Properties["kind"].Enum = []any{"rectangle", "ellipse", "line", "arrow", "text", "freehand"}
			s.Properties["x"].Default, s.Properties["y"].Default, s.Properties["rotation"].Default = 0, 0, 0
		}
		if variant.op == "update" || variant.op == "shape.update" {
			for _, name := range []string{"width", "height", "parentPlacementId"} {
				if s.Properties[name] != nil {
					s.Properties[name].Nullable = true
				}
			}
		}
		if variant.op == "link.override" {
			s.Properties["waypoints"].Nullable = true
		}
		s.PrecomputeMessages()
		r.Map()[variant.name] = s
		ref := "#/components/schemas/" + variant.name
		union.OneOf = append(union.OneOf, &huma.Schema{Ref: ref})
		union.Discriminator.Mapping[variant.op] = ref
	}
	return union
}
