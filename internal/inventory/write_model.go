package inventory

import (
	"encoding/json"

	"github.com/google/uuid"
)

func presentFields(data []byte) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	err := json.Unmarshal(data, &fields)
	return fields, err
}

type PlaceOn struct {
	CanvasID          uuid.UUID      `json:"canvasId" format:"uuid"`
	ParentPlacementID *uuid.UUID     `json:"parentPlacementId,omitempty" required:"false"`
	X                 float64        `json:"x,omitempty" required:"false"`
	Y                 float64        `json:"y,omitempty" required:"false"`
	Width             *float64       `json:"width,omitempty" required:"false"`
	Height            *float64       `json:"height,omitempty" required:"false"`
	Style             map[string]any `json:"style,omitempty" required:"false"`
}

type AreaCreate struct {
	Name        string     `json:"name" minLength:"1" maxLength:"100"`
	ParentID    *uuid.UUID `json:"parentId,omitempty" required:"false"`
	Kind        string     `json:"kind,omitempty" enum:"site,network,vlan,cloud,custom" required:"false"`
	Color       string     `json:"color,omitempty" enum:"yellow,teal,violet,rose,blue,green,gray" required:"false"`
	CIDR        *string    `json:"cidr,omitempty" required:"false"`
	VlanID      *int32     `json:"vlanId,omitempty" minimum:"1" maximum:"4094" required:"false"`
	Description string     `json:"description,omitempty" required:"false"`
	Properties  []Field    `json:"properties,omitempty" required:"false"`
	PlaceOn     *PlaceOn   `json:"placeOn,omitempty" required:"false"`
}

type AreaUpdate struct {
	Name                        *string    `json:"name,omitempty" minLength:"1" maxLength:"100" required:"false"`
	ParentID                    *uuid.UUID `json:"parentId,omitempty" required:"false"`
	Kind                        *string    `json:"kind,omitempty" enum:"site,network,vlan,cloud,custom" required:"false"`
	Color                       *string    `json:"color,omitempty" enum:"yellow,teal,violet,rose,blue,green,gray" required:"false"`
	CIDR                        *string    `json:"cidr,omitempty" required:"false"`
	VlanID                      *int32     `json:"vlanId,omitempty" minimum:"1" maximum:"4094" required:"false"`
	Description                 *string    `json:"description,omitempty" required:"false"`
	Properties                  []Field    `json:"properties,omitempty" required:"false"`
	parentSet, cidrSet, vlanSet bool
}

func (v *AreaUpdate) UnmarshalJSON(data []byte) error {
	type alias AreaUpdate
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	fields, err := presentFields(data)
	if err != nil {
		return err
	}
	*v = AreaUpdate(decoded)
	_, v.parentSet = fields["parentId"]
	_, v.cidrSet = fields["cidr"]
	_, v.vlanSet = fields["vlanId"]
	return nil
}

func (v AreaUpdate) hasChange() bool {
	return v.Name != nil || v.parentSet || v.Kind != nil || v.Color != nil || v.cidrSet || v.vlanSet || v.Description != nil || v.Properties != nil
}

type NodeCreate struct {
	Name        string     `json:"name" minLength:"1" maxLength:"100"`
	Slug        *string    `json:"slug,omitempty" required:"false"`
	Category    string     `json:"category" enum:"network,compute,service,storage,external,generic"`
	Type        string     `json:"type,omitempty" required:"false"`
	Icon        *string    `json:"icon,omitempty" required:"false"`
	Description string     `json:"description,omitempty" required:"false"`
	AreaID      *uuid.UUID `json:"areaId,omitempty" required:"false"`
	Properties  []Field    `json:"properties,omitempty" required:"false"`
	IsProtected bool       `json:"isProtected,omitempty" required:"false"`
	PlaceOn     *PlaceOn   `json:"placeOn,omitempty" required:"false"`
}

type NodeUpdate struct {
	Name             *string    `json:"name,omitempty" minLength:"1" maxLength:"100" required:"false"`
	Slug             *string    `json:"slug,omitempty" required:"false"`
	Category         *string    `json:"category,omitempty" enum:"network,compute,service,storage,external,generic" required:"false"`
	Type             *string    `json:"type,omitempty" required:"false"`
	Icon             *string    `json:"icon,omitempty" required:"false"`
	Description      *string    `json:"description,omitempty" required:"false"`
	AreaID           *uuid.UUID `json:"areaId,omitempty" required:"false"`
	Properties       []Field    `json:"properties,omitempty" required:"false"`
	IsProtected      *bool      `json:"isProtected,omitempty" required:"false"`
	iconSet, areaSet bool
}

func (v *NodeUpdate) UnmarshalJSON(data []byte) error {
	type alias NodeUpdate
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	fields, err := presentFields(data)
	if err != nil {
		return err
	}
	*v = NodeUpdate(decoded)
	_, v.iconSet = fields["icon"]
	_, v.areaSet = fields["areaId"]
	return nil
}

func (v NodeUpdate) hasChange() bool {
	return v.Name != nil || v.Slug != nil || v.Category != nil || v.Type != nil || v.iconSet || v.Description != nil || v.areaSet || v.Properties != nil || v.IsProtected != nil
}

type HealthSet struct {
	Status  string  `json:"status" enum:"healthy,degraded,down,unknown"`
	Message *string `json:"message,omitempty" maxLength:"500" required:"false"`
}

type LinkCreate struct {
	SourceNodeID uuid.UUID `json:"sourceNodeId" format:"uuid"`
	TargetNodeID uuid.UUID `json:"targetNodeId" format:"uuid"`
	Kind         string    `json:"kind" enum:"physical,vpn_tunnel,proxy_route,dns_dependency,storage_mount,logical"`
	Directed     bool      `json:"directed,omitempty" required:"false"`
	Label        string    `json:"label,omitempty" required:"false"`
	SourcePort   *string   `json:"sourcePort,omitempty" required:"false"`
	TargetPort   *string   `json:"targetPort,omitempty" required:"false"`
	Properties   []Field   `json:"properties,omitempty" required:"false"`
}

type LinkUpdate struct {
	Kind                         *string `json:"kind,omitempty" enum:"physical,vpn_tunnel,proxy_route,dns_dependency,storage_mount,logical" required:"false"`
	Directed                     *bool   `json:"directed,omitempty" required:"false"`
	Label                        *string `json:"label,omitempty" required:"false"`
	SourcePort                   *string `json:"sourcePort,omitempty" required:"false"`
	TargetPort                   *string `json:"targetPort,omitempty" required:"false"`
	Properties                   []Field `json:"properties,omitempty" required:"false"`
	sourcePortSet, targetPortSet bool
}

func (v *LinkUpdate) UnmarshalJSON(data []byte) error {
	type alias LinkUpdate
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	fields, err := presentFields(data)
	if err != nil {
		return err
	}
	*v = LinkUpdate(decoded)
	_, v.sourcePortSet = fields["sourcePort"]
	_, v.targetPortSet = fields["targetPort"]
	return nil
}

func (v LinkUpdate) hasChange() bool {
	return v.Kind != nil || v.Directed != nil || v.Label != nil || v.sourcePortSet || v.targetPortSet || v.Properties != nil
}

type NoteCreate struct {
	Target      NoteTarget `json:"target"`
	Kind        string     `json:"kind,omitempty" enum:"note,rule" required:"false"`
	Title       string     `json:"title,omitempty" maxLength:"200" required:"false"`
	Body        string     `json:"body"`
	IsSensitive bool       `json:"isSensitive,omitempty" required:"false"`
	Color       string     `json:"color,omitempty" enum:"default,yellow,teal,violet,rose" required:"false"`
	PlaceOn     *PlaceOn   `json:"placeOn,omitempty" required:"false"`
}

type NoteUpdate struct {
	Kind        *string `json:"kind,omitempty" enum:"note,rule" required:"false"`
	Title       *string `json:"title,omitempty" maxLength:"200" required:"false"`
	Body        *string `json:"body,omitempty" required:"false"`
	IsSensitive *bool   `json:"isSensitive,omitempty" required:"false"`
	Color       *string `json:"color,omitempty" enum:"default,yellow,teal,violet,rose" required:"false"`
}

func (v NoteUpdate) hasChange() bool {
	return v.Kind != nil || v.Title != nil || v.Body != nil || v.IsSensitive != nil || v.Color != nil
}
