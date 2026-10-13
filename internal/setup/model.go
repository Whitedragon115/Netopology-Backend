package setup

// SetupStatus is the response of getSetupStatus (contract schema SetupStatus).
type SetupStatus struct {
	NeedsSetup bool `json:"needsSetup"`
}

// SetupRequest is the body of completeSetup (contract schema SetupRequest).
// Huma validates the tags before the handler runs, so the service can trust lengths and patterns.
type SetupRequest struct {
	Username      string `json:"username" pattern:"^[A-Za-z0-9][A-Za-z0-9._-]{1,38}[A-Za-z0-9]$" example:"alice"`
	Password      string `json:"password" minLength:"12" maxLength:"256" writeOnly:"true"`
	DisplayName   string `json:"displayName,omitempty" maxLength:"100"`
	Email         string `json:"email,omitempty" format:"email"`
	WorkspaceName string `json:"workspaceName,omitempty" maxLength:"100" doc:"If set, also create this first workspace with you as owner"`
}
