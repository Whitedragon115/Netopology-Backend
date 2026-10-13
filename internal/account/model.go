package account

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

type User struct {
	ID          uuid.UUID `json:"id" format:"uuid"`
	Username    string    `json:"username" pattern:"^[A-Za-z0-9][A-Za-z0-9._-]{1,38}[A-Za-z0-9]$"`
	DisplayName string    `json:"displayName"`
}

type MeUpdate struct {
	DisplayName *string `json:"displayName,omitempty" maxLength:"100"`
	Email       *string `json:"email" format:"email" required:"false"`
	EmailSet    bool    `json:"-"`
}

func (m *MeUpdate) UnmarshalJSON(data []byte) error {
	type alias MeUpdate
	var value alias
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*m = MeUpdate(value)
	_, m.EmailSet = fields["email"]
	return nil
}

type PasswordChange struct {
	CurrentPassword string `json:"currentPassword" writeOnly:"true"`
	NewPassword     string `json:"newPassword" minLength:"12" maxLength:"256" writeOnly:"true"`
}

type PasswordReset struct {
	NewPassword string `json:"newPassword" minLength:"12" maxLength:"256" writeOnly:"true"`
}

type Identity struct {
	ID          uuid.UUID  `json:"id" format:"uuid"`
	Provider    string     `json:"provider" enum:"google,discord"`
	Email       *string    `json:"email" required:"false"`
	DisplayName *string    `json:"displayName" required:"false"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt" required:"false"`
}

func identityFrom(row dbgen.UserIdentity) Identity {
	return Identity{
		ID: row.ID, Provider: row.Provider, Email: row.Email, DisplayName: row.DisplayName,
		CreatedAt: row.CreatedAt, LastUsedAt: row.LastUsedAt,
	}
}

type AccountCreate struct {
	Username     string  `json:"username" pattern:"^[A-Za-z0-9][A-Za-z0-9._-]{1,38}[A-Za-z0-9]$"`
	Password     *string `json:"password,omitempty" minLength:"12" maxLength:"256" writeOnly:"true"`
	DisplayName  string  `json:"displayName,omitempty" maxLength:"100"`
	Email        *string `json:"email,omitempty" format:"email"`
	InstanceRole string  `json:"instanceRole,omitempty" enum:"admin,member" default:"member"`
}

type AccountUpdate struct {
	DisplayName  *string `json:"displayName,omitempty" maxLength:"100"`
	Email        *string `json:"email" format:"email" required:"false"`
	Status       *string `json:"status,omitempty" enum:"active,disabled"`
	InstanceRole *string `json:"instanceRole,omitempty" enum:"owner,admin,member"`
	EmailSet     bool    `json:"-"`
}

func (a *AccountUpdate) UnmarshalJSON(data []byte) error {
	type alias AccountUpdate
	var value alias
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*a = AccountUpdate(value)
	_, a.EmailSet = fields["email"]
	return nil
}

type AccountPage struct {
	Items      []apimodel.Account `json:"items"`
	NextCursor *string            `json:"nextCursor" required:"false"`
}

type InstanceSettings struct {
	SessionTtlHours      *int  `json:"sessionTtlHours" minimum:"1" maximum:"8760" default:"168" required:"false"`
	OauthAutoLinkByEmail *bool `json:"oauthAutoLinkByEmail" default:"true" required:"false"`
}
