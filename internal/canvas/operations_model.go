package canvas

import (
	"encoding/json"

	"github.com/google/uuid"
)

type CanvasOperationBatch struct {
	Operations []CanvasOperation `json:"operations" minItems:"1" maxItems:"500"`
}

// CanvasOperation accepts the contract's eight discriminated operation variants.
type CanvasOperation struct {
	Op                                           string         `json:"op" enum:"place,update,remove,shape.create,shape.update,shape.delete,link.override,link.reset"`
	ClientID                                     *string        `json:"clientId,omitempty" required:"false"`
	Item                                         *ItemRef       `json:"item,omitempty" required:"false"`
	PlacementID                                  *uuid.UUID     `json:"placementId,omitempty" required:"false"`
	ShapeID                                      *uuid.UUID     `json:"shapeId,omitempty" required:"false"`
	LinkID                                       *uuid.UUID     `json:"linkId,omitempty" required:"false"`
	ParentPlacementID                            *uuid.UUID     `json:"parentPlacementId,omitempty" required:"false"`
	Kind                                         *string        `json:"kind,omitempty" required:"false"`
	X                                            *float64       `json:"x,omitempty" required:"false"`
	Y                                            *float64       `json:"y,omitempty" required:"false"`
	Width                                        *float64       `json:"width,omitempty" required:"false"`
	Height                                       *float64       `json:"height,omitempty" required:"false"`
	Rotation                                     *float64       `json:"rotation,omitempty" required:"false"`
	ZIndex                                       *int32         `json:"zIndex,omitempty" required:"false"`
	Collapsed                                    *bool          `json:"collapsed,omitempty" required:"false"`
	Hidden                                       *bool          `json:"hidden,omitempty" required:"false"`
	Style                                        map[string]any `json:"style,omitempty" required:"false"`
	Data                                         map[string]any `json:"data,omitempty" required:"false"`
	Waypoints                                    []Point        `json:"waypoints,omitempty" required:"false"`
	parentSet, widthSet, heightSet, waypointsSet bool
}

func (v *CanvasOperation) UnmarshalJSON(data []byte) error {
	type alias CanvasOperation
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*v = CanvasOperation(decoded)
	_, v.parentSet = fields["parentPlacementId"]
	_, v.widthSet = fields["width"]
	_, v.heightSet = fields["height"]
	_, v.waypointsSet = fields["waypoints"]
	return nil
}

type CanvasOperationResultItem struct {
	Index               int         `json:"index"`
	ClientID            *string     `json:"clientId,omitempty" required:"false"`
	ID                  *uuid.UUID  `json:"id,omitempty" required:"false"`
	RemovedPlacementIDs []uuid.UUID `json:"removedPlacementIds,omitempty" required:"false"`
}

type CanvasOperationResult struct {
	Results []CanvasOperationResultItem `json:"results"`
	View    *CanvasView                 `json:"view,omitempty" required:"false"`
}
