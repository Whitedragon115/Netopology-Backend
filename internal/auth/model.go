package auth

// LoginRequest is the contract's username/password sign-in body.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password" writeOnly:"true"`
	Remember bool   `json:"remember,omitempty" default:"false"`
}

type ProviderOption struct {
	ID   string `json:"id" enum:"google,discord"`
	Name string `json:"name"`
}

type AuthProviders struct {
	Password  bool             `json:"password"`
	Providers []ProviderOption `json:"providers"`
}
