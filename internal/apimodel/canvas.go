package apimodel

import (
	"time"

	"github.com/google/uuid"
)

type (
	CanvasViewport struct {
		X    float64 `json:"x"`
		Y    float64 `json:"y"`
		Zoom float64 `json:"zoom"`
	}
	CanvasSettings map[string]any
	CanvasLock     struct {
		UserID   uuid.UUID `json:"userId"`
		LockedAt time.Time `json:"lockedAt"`
	}
	Canvas struct {
		ID             uuid.UUID      `json:"id"`
		Kind           string         `json:"kind" enum:"tab,area"`
		AreaID         *uuid.UUID     `json:"areaId"`
		Name           string         `json:"name" maxLength:"100"`
		Description    string         `json:"description"`
		SortOrder      int32          `json:"sortOrder"`
		CanvasViewport CanvasViewport `json:"viewport"`
		CanvasSettings CanvasSettings `json:"settings"`
		IsProtected    bool           `json:"isProtected"`
		CanvasLock     *CanvasLock    `json:"lock"`
		CreatedAt      time.Time      `json:"createdAt"`
		UpdatedAt      time.Time      `json:"updatedAt"`
		Version        int32          `json:"version"`
	}
)

type (
	CanvasPage struct {
		Items []Canvas `json:"items"`
	}
	CanvasCreate struct {
		Name           string         `json:"name" minLength:"1" maxLength:"100"`
		Description    string         `json:"description,omitempty" required:"false"`
		SortOrder      int32          `json:"sortOrder,omitempty" required:"false"`
		CanvasSettings CanvasSettings `json:"settings,omitempty" required:"false"`
		IsProtected    bool           `json:"isProtected,omitempty" required:"false"`
	}
)

type CanvasUpdate struct {
	Name           *string         `json:"name,omitempty" minLength:"1" maxLength:"100" required:"false"`
	Description    *string         `json:"description,omitempty" required:"false"`
	SortOrder      *int32          `json:"sortOrder,omitempty" required:"false"`
	CanvasViewport *CanvasViewport `json:"viewport,omitempty" required:"false"`
	CanvasSettings CanvasSettings  `json:"settings,omitempty" required:"false"`
	IsProtected    *bool           `json:"isProtected,omitempty" required:"false"`
}
type (
	ItemRef struct {
		Type string    `json:"type" enum:"node,area,note"`
		ID   uuid.UUID `json:"id"`
	}
	Placement struct {
		ID                uuid.UUID      `json:"id"`
		Item              ItemRef        `json:"item"`
		ParentPlacementID *uuid.UUID     `json:"parentPlacementId"`
		X                 float64        `json:"x"`
		Y                 float64        `json:"y"`
		Width             *float64       `json:"width,omitempty" required:"false"`
		Height            *float64       `json:"height,omitempty" required:"false"`
		ZIndex            int32          `json:"zIndex"`
		Collapsed         bool           `json:"collapsed,omitempty" required:"false"`
		Style             map[string]any `json:"style"`
	}
)

type LinkOverride struct {
	LinkID    uuid.UUID      `json:"linkId"`
	Hidden    bool           `json:"hidden"`
	Style     map[string]any `json:"style"`
	Waypoints []Point        `json:"waypoints,omitempty" required:"false"`
}
type (
	Point struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	BreadcrumbItem struct {
		AreaID   *uuid.UUID `json:"areaId"`
		Name     string     `json:"name"`
		CanvasID uuid.UUID  `json:"canvasId"`
	}
	Shape struct {
		ID       uuid.UUID      `json:"id"`
		Kind     string         `json:"kind"`
		X        float64        `json:"x"`
		Y        float64        `json:"y"`
		Width    *float64       `json:"width,omitempty" required:"false"`
		Height   *float64       `json:"height,omitempty" required:"false"`
		Rotation float64        `json:"rotation"`
		ZIndex   int32          `json:"zIndex"`
		Style    map[string]any `json:"style"`
		Data     map[string]any `json:"data"`
	}
)

type (
	Viewer struct {
		Role             string `json:"role"`
		CanEdit          bool   `json:"canEdit"`
		CanViewSensitive bool   `json:"canViewSensitive"`
	}
	CanvasView struct {
		Canvas        Canvas           `json:"canvas"`
		Viewer        Viewer           `json:"viewer"`
		Breadcrumb    []BreadcrumbItem `json:"breadcrumb"`
		Placements    []Placement      `json:"placements"`
		LinkOverrides []LinkOverride   `json:"linkOverrides"`
		Nodes         []Node           `json:"nodes"`
		Links         []Link           `json:"links"`
		Areas         []Area           `json:"areas"`
		Notes         []Note           `json:"notes"`
		Shapes        []Shape          `json:"shapes"`
	}
)
