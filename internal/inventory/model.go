package inventory

import (
	"time"

	"github.com/google/uuid"
)

type Field struct {
	Key       string  `json:"key" pattern:"^[a-z][a-z0-9_]{0,39}$"`
	Label     string  `json:"label" maxLength:"60"`
	Value     *string `json:"value,omitempty" maxLength:"2000" required:"false"`
	Sensitive bool    `json:"sensitive,omitempty" required:"false"`
	Redacted  bool    `json:"redacted,omitempty" required:"false"`
}

type Health struct {
	Status    string     `json:"status" enum:"healthy,degraded,down,unknown"`
	Source    string     `json:"source" enum:"manual,plugin"`
	Message   *string    `json:"message,omitempty" required:"false"`
	CheckedAt *time.Time `json:"checkedAt,omitempty" required:"false"`
}

type Area struct {
	ID          uuid.UUID  `json:"id" format:"uuid"`
	ParentID    *uuid.UUID `json:"parentId"`
	Name        string     `json:"name" maxLength:"100"`
	Kind        string     `json:"kind" enum:"site,network,vlan,cloud,custom"`
	Color       string     `json:"color" enum:"yellow,teal,violet,rose,blue,green,gray" required:"false"`
	NodeCount   int64      `json:"nodeCount,omitempty" required:"false"`
	Health      *string    `json:"health,omitempty" required:"false"`
	CIDR        *string    `json:"cidr" required:"false"`
	VlanID      *int32     `json:"vlanId" required:"false"`
	Description string     `json:"description"`
	Properties  []Field    `json:"properties"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	Version     int32      `json:"version"`
}

type Node struct {
	ID          uuid.UUID  `json:"id" format:"uuid"`
	Slug        string     `json:"slug" pattern:"^[a-z0-9][a-z0-9-]{0,62}$"`
	Name        string     `json:"name" maxLength:"100"`
	Category    string     `json:"category" enum:"network,compute,service,storage,external,generic"`
	Type        string     `json:"type"`
	Icon        *string    `json:"icon,omitempty" required:"false"`
	Description string     `json:"description"`
	AreaID      *uuid.UUID `json:"areaId"`
	Properties  []Field    `json:"properties"`
	Health      Health     `json:"health"`
	IsProtected bool       `json:"isProtected"`
	CreatedBy   *uuid.UUID `json:"createdBy,omitempty" required:"false"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	Version     int32      `json:"version"`
}

type Link struct {
	ID           uuid.UUID `json:"id" format:"uuid"`
	SourceNodeID uuid.UUID `json:"sourceNodeId" format:"uuid"`
	TargetNodeID uuid.UUID `json:"targetNodeId" format:"uuid"`
	Kind         string    `json:"kind" enum:"physical,vpn_tunnel,proxy_route,dns_dependency,storage_mount,logical"`
	Directed     bool      `json:"directed"`
	Label        string    `json:"label"`
	SourcePort   *string   `json:"sourcePort,omitempty" required:"false"`
	TargetPort   *string   `json:"targetPort,omitempty" required:"false"`
	Properties   []Field   `json:"properties"`
	Health       Health    `json:"health"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Version      int32     `json:"version"`
}

type NoteTarget struct {
	Type string     `json:"type" enum:"node,link,area,canvas,workspace"`
	ID   *uuid.UUID `json:"id,omitempty" required:"false"`
}

type Note struct {
	ID          uuid.UUID  `json:"id" format:"uuid"`
	Target      NoteTarget `json:"target"`
	Kind        string     `json:"kind" enum:"note,rule"`
	Title       string     `json:"title" maxLength:"200"`
	Body        string     `json:"body"`
	IsSensitive bool       `json:"isSensitive"`
	Redacted    bool       `json:"redacted,omitempty" required:"false"`
	Color       string     `json:"color" enum:"default,yellow,teal,violet,rose"`
	CreatedBy   *uuid.UUID `json:"createdBy,omitempty" required:"false"`
	UpdatedBy   *uuid.UUID `json:"updatedBy,omitempty" required:"false"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	Version     int32      `json:"version"`
}

type (
	AreaPage struct {
		Items      []Area  `json:"items"`
		NextCursor *string `json:"nextCursor,omitempty" required:"false"`
	}
	NodePage struct {
		Items      []Node  `json:"items"`
		NextCursor *string `json:"nextCursor,omitempty" required:"false"`
	}
	LinkPage struct {
		Items      []Link  `json:"items"`
		NextCursor *string `json:"nextCursor,omitempty" required:"false"`
	}
	NotePage struct {
		Items      []Note  `json:"items"`
		NextCursor *string `json:"nextCursor,omitempty" required:"false"`
	}
)

type (
	NodeUsageCanvas struct {
		ID   uuid.UUID `json:"id" format:"uuid"`
		Name string    `json:"name"`
	}
	NodeUsage struct {
		Canvases          []NodeUsageCanvas `json:"canvases"`
		HiddenCanvasCount int64             `json:"hiddenCanvasCount"`
		LinkCount         int64             `json:"linkCount"`
		NoteCount         int64             `json:"noteCount"`
	}
)

// IDs bounds a reusable inventory read to items requested by a canvas or event.
type (
	IDs      struct{ Areas, Nodes, Links, Notes []uuid.UUID }
	Snapshot struct {
		Areas []Area
		Nodes []Node
		Links []Link
		Notes []Note
	}
)
