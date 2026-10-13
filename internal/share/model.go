package share

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

type ShareLink struct {
	ID          uuid.UUID  `json:"id"`
	CanvasID    uuid.UUID  `json:"canvasId"`
	Label       string     `json:"label" maxLength:"100"`
	HasPassword bool       `json:"hasPassword"`
	ExpiresAt   *time.Time `json:"expiresAt" required:"false"`
	RevokedAt   *time.Time `json:"revokedAt" required:"false"`
	CreatedBy   *uuid.UUID `json:"createdBy" required:"false"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt" required:"false"`
	UseCount    int64      `json:"useCount"`
}
type (
	ShareLinkCreate struct {
		Label     string     `json:"label,omitempty" maxLength:"100"`
		Password  *string    `json:"password,omitempty" minLength:"8" maxLength:"256" writeOnly:"true"`
		ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	}
	ShareLinkCreated struct {
		ShareLink ShareLink `json:"shareLink"`
		Token     string    `json:"token"`
		URL       string    `json:"url" format:"uri"`
	}
	ShareLinkUpdate struct {
		Label       *string    `json:"label,omitempty" maxLength:"100"`
		Password    *string    `json:"password" required:"false" minLength:"8" writeOnly:"true"`
		ExpiresAt   *time.Time `json:"expiresAt" required:"false"`
		PasswordSet bool       `json:"-"`
		ExpirySet   bool       `json:"-"`
	}
)

func (p *ShareLinkUpdate) UnmarshalJSON(data []byte) error {
	type alias ShareLinkUpdate
	var value alias
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*p = ShareLinkUpdate(value)
	_, p.PasswordSet = fields["password"]
	_, p.ExpirySet = fields["expiresAt"]
	return nil
}

type (
	ShareInfo struct {
		RequiresPassword bool `json:"requiresPassword"`
	}
	ShareUnlock struct {
		Password string `json:"password,omitempty" writeOnly:"true"`
	}
	ShareLinkPage struct {
		Items []ShareLink `json:"items"`
	}
)

func fromRow(row dbgen.ShareLink) ShareLink {
	return ShareLink{ID: row.ID, CanvasID: row.CanvasID, Label: row.Label, HasPassword: row.PasswordHash != nil, ExpiresAt: row.ExpiresAt, RevokedAt: row.RevokedAt, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, LastUsedAt: row.LastUsedAt, UseCount: row.UseCount}
}
