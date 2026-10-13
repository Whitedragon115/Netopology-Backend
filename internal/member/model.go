package member

import (
	"time"

	"github.com/google/uuid"
)

type PublicUser struct {
	ID          uuid.UUID `json:"id" format:"uuid"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
}

type Member struct {
	User             PublicUser   `json:"user"`
	Role             string       `json:"role" enum:"owner,admin,editor,viewer"`
	CanViewSensitive *bool        `json:"canViewSensitive,omitempty" required:"false"`
	CanvasIDs        *[]uuid.UUID `json:"canvasIds,omitempty" required:"false"`
	CreatedAt        *time.Time   `json:"createdAt,omitempty" required:"false"`
}

type MemberAdd struct {
	UserID           uuid.UUID   `json:"userId" format:"uuid"`
	Role             string      `json:"role" enum:"owner,admin,editor,viewer"`
	CanViewSensitive bool        `json:"canViewSensitive,omitempty" default:"false"`
	CanvasIDs        []uuid.UUID `json:"canvasIds,omitempty"`
}

type MemberUpdate struct {
	Role             *string `json:"role,omitempty" enum:"owner,admin,editor,viewer"`
	CanViewSensitive *bool   `json:"canViewSensitive,omitempty"`
}

type CanvasGrants struct {
	CanvasIDs []uuid.UUID `json:"canvasIds"`
}

type MemberPage struct {
	Items []Member `json:"items"`
}
