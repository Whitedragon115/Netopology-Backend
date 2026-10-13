// Package apimodel holds the wire types (request/response bodies) used by MORE THAN ONE feature.
// A type used by one feature lives in that feature's model.go instead.
//
// Go type names become OpenAPI schema names, so they must match api/contract.yaml
// (components.schemas). Field json names are camelCase. Mapping the contract to Huma tags:
//   - in `required`, not nullable      -> plain field                    (string)
//   - in `required`, nullable          -> pointer                        (*string)
//   - not required, nullable           -> pointer + `required:"false"`   (always sent, maybe null)
//   - not required, not nullable       -> `json:",omitempty"`            (left out when empty)
package apimodel

import (
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

// Account is the full account, as the account itself or the server owner/admin sees it.
// (The contract's `Me` is an alias of Account.)
type Account struct {
	ID           uuid.UUID  `json:"id" format:"uuid"`
	Username     string     `json:"username"`
	DisplayName  string     `json:"displayName"`
	Email        *string    `json:"email" format:"email" required:"false"`
	InstanceRole string     `json:"instanceRole" enum:"owner,admin,member"`
	HasPassword  bool       `json:"hasPassword"`
	Identities   []string   `json:"identities" enum:"google,discord" doc:"Linked sign-in providers"`
	Status       string     `json:"status" enum:"active,disabled"`
	LastLoginAt  *time.Time `json:"lastLoginAt" required:"false"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// AccountFrom maps a users row plus its linked providers.
func AccountFrom(u dbgen.User, providers []string) Account {
	if providers == nil {
		providers = []string{}
	}
	return Account{
		ID:           u.ID,
		Username:     u.Username,
		DisplayName:  u.DisplayName,
		Email:        u.Email,
		InstanceRole: u.InstanceRole,
		HasPassword:  u.PasswordHash != nil,
		Identities:   providers,
		Status:       u.Status,
		LastLoginAt:  u.LastLoginAt,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}
