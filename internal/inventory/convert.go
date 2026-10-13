package inventory

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

func areaBase(id, wsID uuid.UUID, parentID *uuid.UUID, name, kind, color string,
	cidr *netip.Prefix, vlanID *int32, description string, fields json.RawMessage,
	createdBy *uuid.UUID, createdAt, updatedAt time.Time, version int32,
) dbgen.Area {
	return dbgen.Area{
		ID: id, WorkspaceID: wsID, ParentID: parentID, Name: name, Kind: kind, Color: color,
		Cidr: cidr, VlanID: vlanID, Description: description, Properties: fields,
		CreatedBy: createdBy, CreatedAt: createdAt, UpdatedAt: updatedAt, Version: version,
	}
}

func properties(raw json.RawMessage) ([]Field, error) {
	fields := []Field{}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("decode inventory properties: %w", err)
	}
	return fields, nil
}

func areaFromRow(row dbgen.Area, nodeCount int64, audience Audience) (Area, error) {
	fields, err := properties(row.Properties)
	if err != nil {
		return Area{}, err
	}
	var cidr *string
	if row.Cidr != nil {
		value := row.Cidr.String()
		cidr = &value
	}
	area := Area{
		ID: row.ID, ParentID: row.ParentID, Name: row.Name, Kind: row.Kind, Color: row.Color,
		NodeCount: nodeCount, CIDR: cidr, VlanID: row.VlanID, Description: row.Description,
		Properties: fields, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Version: row.Version,
	}
	redactArea(&area, audience)
	return area, nil
}

func nodeFromRow(row dbgen.Node, audience Audience) (Node, error) {
	fields, err := properties(row.Properties)
	if err != nil {
		return Node{}, err
	}
	node := Node{
		ID: row.ID, Slug: row.Slug, Name: row.Name, Category: row.Category, Type: row.Type,
		Icon: row.Icon, Description: row.Description, AreaID: row.AreaID, Properties: fields,
		Health:      Health{Status: row.HealthStatus, Source: row.HealthSource, Message: row.HealthMessage, CheckedAt: row.HealthCheckedAt},
		IsProtected: row.IsProtected, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt, Version: row.Version,
	}
	redactNode(&node, audience)
	return node, nil
}

func linkFromRow(row dbgen.Link, audience Audience) (Link, error) {
	fields, err := properties(row.Properties)
	if err != nil {
		return Link{}, err
	}
	link := Link{
		ID: row.ID, SourceNodeID: row.SourceNodeID, TargetNodeID: row.TargetNodeID,
		Kind: row.Kind, Directed: row.Directed, Label: row.Label, SourcePort: row.SourcePort,
		TargetPort: row.TargetPort, Properties: fields,
		Health:    Health{Status: row.HealthStatus, Source: row.HealthSource, Message: row.HealthMessage, CheckedAt: row.HealthCheckedAt},
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Version: row.Version,
	}
	redactLink(&link, audience)
	return link, nil
}

func noteFromRow(row dbgen.Note, audience Audience) (Note, bool) {
	target := NoteTarget{Type: "workspace"}
	switch {
	case row.NodeID != nil:
		target = NoteTarget{Type: "node", ID: row.NodeID}
	case row.LinkID != nil:
		target = NoteTarget{Type: "link", ID: row.LinkID}
	case row.AreaID != nil:
		target = NoteTarget{Type: "area", ID: row.AreaID}
	case row.CanvasID != nil:
		target = NoteTarget{Type: "canvas", ID: row.CanvasID}
	}
	note := Note{
		ID: row.ID, Target: target, Kind: row.Kind, Title: row.Title, Body: row.Body,
		IsSensitive: row.IsSensitive, Color: row.Color, CreatedBy: row.CreatedBy,
		UpdatedBy: row.UpdatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Version: row.Version,
	}
	if !redactNote(&note, audience) {
		return Note{}, false
	}
	return note, true
}
