package workspace

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

type Membership struct {
	Role             string `json:"role" enum:"owner,admin,editor,viewer"`
	CanViewSensitive bool   `json:"canViewSensitive"`
}

type Workspace struct {
	ID          uuid.UUID      `json:"id" format:"uuid"`
	Name        string         `json:"name" maxLength:"100"`
	Slug        string         `json:"slug" pattern:"^[a-z0-9][a-z0-9-]{0,62}$"`
	Description string         `json:"description"`
	IsProtected bool           `json:"isProtected"`
	Settings    map[string]any `json:"settings"`
	Membership  Membership     `json:"membership"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	Version     int32          `json:"version"`
}

type WorkspaceCreate struct {
	Name        string `json:"name" minLength:"1" maxLength:"100"`
	Slug        string `json:"slug,omitempty" pattern:"^[a-z0-9][a-z0-9-]{0,62}$"`
	Description string `json:"description,omitempty"`
}

type WorkspaceUpdate struct {
	Name        *string        `json:"name,omitempty" minLength:"1" maxLength:"100"`
	Slug        *string        `json:"slug,omitempty" pattern:"^[a-z0-9][a-z0-9-]{0,62}$"`
	Description *string        `json:"description,omitempty"`
	IsProtected *bool          `json:"isProtected,omitempty"`
	Settings    map[string]any `json:"settings,omitempty"`
}

type Page struct {
	Items []Workspace `json:"items"`
}

func fromRow(row dbgen.Workspace, member Membership) (Workspace, error) {
	settings := map[string]any{}
	if err := json.Unmarshal(row.Settings, &settings); err != nil {
		return Workspace{}, err
	}
	return Workspace{
		ID: row.ID, Name: row.Name, Slug: row.Slug, Description: row.Description,
		IsProtected: row.IsProtected, Settings: settings, Membership: member,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Version: row.Version,
	}, nil
}
