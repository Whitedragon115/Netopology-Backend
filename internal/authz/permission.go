// Package authz is the single place that says who may do what.
//
// Source of truth: docs/design/permissions.md. Every API operation declares one Permission
// (its OpenAPI x-permission); the same key is written to the audit log. The contract test in
// internal/server fails if an operation's key is missing here or differs from the contract.
//
// This file answers "may this ROLE use this permission at all". The finer rules (editor
// visibility set, protected items, owners vs admins, last owner, self-only removal, ...) are
// enforced by the feature services; each is listed in the "Notes" column of permissions.md §3
// and marked "service rule" below.
package authz

// Permission is a permission key such as "node.update".
type Permission string

// Scope says what kind of caller a permission is for.
type Scope int

const (
	ScopeAnonymous Scope = iota // anyone, no cookie needed
	ScopeAccount                // any signed-in account
	ScopeInstance               // server owner / server admin (instance_role)
	ScopeWorkspace              // a member of the workspace in the path, with one of the roles
	ScopeGuest                  // a share-link guest (netopology_share cookie)
)

type roleSet uint8

const (
	o roleSet = 1 << iota // owner
	a                     // admin
	e                     // editor
	v                     // viewer
)

type rule struct {
	scope Scope
	roles roleSet // ScopeWorkspace only
}

// Setup, session, profile
const (
	SetupStatus      Permission = "setup.status"
	SetupComplete    Permission = "setup.complete"
	AuthProviders    Permission = "auth.providers"
	AuthLogin        Permission = "auth.login"
	AuthOAuthStart   Permission = "auth.oauth_start"
	AuthOAuthCB      Permission = "auth.oauth_callback"
	AuthLogout       Permission = "auth.logout"
	MeRead           Permission = "me.read"
	MeUpdate         Permission = "me.update"
	MePassword       Permission = "me.password"
	MeIdentityList   Permission = "me.identity_list"
	MeIdentityLink   Permission = "me.identity_link"
	MeIdentityUnlink Permission = "me.identity_unlink"
)

// Instance
const (
	UserList              Permission = "user.list"
	UserRead              Permission = "user.read"
	UserCreate            Permission = "user.create"
	UserUpdate            Permission = "user.update"
	UserPasswordReset     Permission = "user.password_reset"
	UserLookup            Permission = "user.lookup"
	InstanceSettingsRead  Permission = "instance.settings_read"
	InstanceSettingsWrite Permission = "instance.settings_update"
)

// Workspaces, members, audit
const (
	WorkspaceList   Permission = "workspace.list"
	WorkspaceCreate Permission = "workspace.create"
	WorkspaceRead   Permission = "workspace.read"
	WorkspaceUpdate Permission = "workspace.update"
	WorkspaceDelete Permission = "workspace.delete"
	MemberList      Permission = "member.list"
	MemberAdd       Permission = "member.add"
	MemberUpdate    Permission = "member.update"
	MemberRemove    Permission = "member.remove"
	MemberGrantsSet Permission = "member.grants_set"
	AuditList       Permission = "audit.list"
)

// Inventory
const (
	AreaList      Permission = "area.list"
	AreaRead      Permission = "area.read"
	AreaCreate    Permission = "area.create"
	AreaUpdate    Permission = "area.update"
	AreaDelete    Permission = "area.delete"
	AreaCanvas    Permission = "area.canvas"
	NodeList      Permission = "node.list"
	NodeRead      Permission = "node.read"
	NodeUsage     Permission = "node.usage"
	NodeCreate    Permission = "node.create"
	NodeUpdate    Permission = "node.update"
	NodeDelete    Permission = "node.delete"
	NodeHealthSet Permission = "node.health_set"
	LinkList      Permission = "link.list"
	LinkRead      Permission = "link.read"
	LinkCreate    Permission = "link.create"
	LinkUpdate    Permission = "link.update"
	LinkDelete    Permission = "link.delete"
	NoteList      Permission = "note.list"
	NoteRead      Permission = "note.read"
	NoteCreate    Permission = "note.create"
	NoteUpdate    Permission = "note.update"
	NoteDelete    Permission = "note.delete"
)

// Canvases, sharing, events
const (
	CanvasList      Permission = "canvas.list"
	CanvasRead      Permission = "canvas.read"
	CanvasView      Permission = "canvas.view"
	CanvasCreate    Permission = "canvas.create"
	CanvasUpdate    Permission = "canvas.update"
	CanvasDelete    Permission = "canvas.delete"
	CanvasEdit      Permission = "canvas.edit"
	CanvasLock      Permission = "canvas.lock"
	CanvasUnlock    Permission = "canvas.unlock"
	ShareLinkList   Permission = "share_link.list"
	ShareLinkCreate Permission = "share_link.create"
	ShareLinkUpdate Permission = "share_link.update"
	ShareLinkRevoke Permission = "share_link.revoke"
	ShareInfo       Permission = "share.info"
	ShareUnlock     Permission = "share.unlock"
	ShareView       Permission = "share.view"
	EventsStream    Permission = "events.stream"
)

var rules = map[Permission]rule{
	SetupStatus:      {scope: ScopeAnonymous},
	SetupComplete:    {scope: ScopeAnonymous}, // service rule: only while no account exists
	AuthProviders:    {scope: ScopeAnonymous},
	AuthLogin:        {scope: ScopeAnonymous}, // service rule: rate-limited per IP and username
	AuthOAuthStart:   {scope: ScopeAnonymous},
	AuthOAuthCB:      {scope: ScopeAnonymous}, // service rule: linked identities only (rule 10)
	AuthLogout:       {scope: ScopeAccount},
	MeRead:           {scope: ScopeAccount},
	MeUpdate:         {scope: ScopeAccount},
	MePassword:       {scope: ScopeAccount},
	MeIdentityList:   {scope: ScopeAccount},
	MeIdentityLink:   {scope: ScopeAccount},
	MeIdentityUnlink: {scope: ScopeAccount}, // service rule: not the last sign-in method

	UserList:              {scope: ScopeInstance},
	UserRead:              {scope: ScopeInstance},
	UserCreate:            {scope: ScopeInstance}, // service rule: only SO creates server admins
	UserUpdate:            {scope: ScopeInstance}, // service rule: SA can't touch SO/SA or grant admin
	UserPasswordReset:     {scope: ScopeInstance}, // service rule: SA can't reset SO/SA; ends sessions
	UserLookup:            {scope: ScopeAccount},  // service rule: caller must be owner/admin of SOME workspace
	InstanceSettingsRead:  {scope: ScopeInstance},
	InstanceSettingsWrite: {scope: ScopeInstance},

	WorkspaceList:   {scope: ScopeAccount},  // only workspaces you are a member of
	WorkspaceCreate: {scope: ScopeInstance}, // creator becomes the workspace owner
	WorkspaceRead:   {scope: ScopeWorkspace, roles: o | a | e | v},
	WorkspaceUpdate: {scope: ScopeWorkspace, roles: o | a}, // service rule: admin can't clear isProtected
	WorkspaceDelete: {scope: ScopeWorkspace, roles: o},     // service rule: not while protected
	MemberList:      {scope: ScopeWorkspace, roles: o | a | e | v},
	MemberAdd:       {scope: ScopeWorkspace, roles: o | a},         // service rule: admin can't add owners
	MemberUpdate:    {scope: ScopeWorkspace, roles: o | a},         // service rule: admin not on owners, can't grant owner
	MemberRemove:    {scope: ScopeWorkspace, roles: o | a | e | v}, // service rule: E/V self only; not last owner
	MemberGrantsSet: {scope: ScopeWorkspace, roles: o | a},
	AuditList:       {scope: ScopeWorkspace, roles: o | a},

	AreaList:      {scope: ScopeWorkspace, roles: o | a | e | v}, // E: visibility set
	AreaRead:      {scope: ScopeWorkspace, roles: o | a | e | v},
	AreaCreate:    {scope: ScopeWorkspace, roles: o | a | e}, // E: placeOn a granted canvas
	AreaUpdate:    {scope: ScopeWorkspace, roles: o | a | e},
	AreaDelete:    {scope: ScopeWorkspace, roles: o | a | e},     // E: rule 5
	AreaCanvas:    {scope: ScopeWorkspace, roles: o | a | e | v}, // rule 11
	NodeList:      {scope: ScopeWorkspace, roles: o | a | e | v},
	NodeRead:      {scope: ScopeWorkspace, roles: o | a | e | v},
	NodeUsage:     {scope: ScopeWorkspace, roles: o | a | e | v},
	NodeCreate:    {scope: ScopeWorkspace, roles: o | a | e},
	NodeUpdate:    {scope: ScopeWorkspace, roles: o | a | e}, // rule 6 for isProtected
	NodeDelete:    {scope: ScopeWorkspace, roles: o | a | e}, // rule 5, rule 6
	NodeHealthSet: {scope: ScopeWorkspace, roles: o | a | e},
	LinkList:      {scope: ScopeWorkspace, roles: o | a | e | v},
	LinkRead:      {scope: ScopeWorkspace, roles: o | a | e | v},
	LinkCreate:    {scope: ScopeWorkspace, roles: o | a | e}, // E: both ends visible
	LinkUpdate:    {scope: ScopeWorkspace, roles: o | a | e},
	LinkDelete:    {scope: ScopeWorkspace, roles: o | a | e},
	NoteList:      {scope: ScopeWorkspace, roles: o | a | e | v}, // redaction §4
	NoteRead:      {scope: ScopeWorkspace, roles: o | a | e | v},
	NoteCreate:    {scope: ScopeWorkspace, roles: o | a | e}, // E: not workspace-level notes
	NoteUpdate:    {scope: ScopeWorkspace, roles: o | a | e},
	NoteDelete:    {scope: ScopeWorkspace, roles: o | a | e},

	CanvasList:      {scope: ScopeWorkspace, roles: o | a | e | v}, // E: granted only
	CanvasRead:      {scope: ScopeWorkspace, roles: o | a | e | v},
	CanvasView:      {scope: ScopeWorkspace, roles: o | a | e | v},
	CanvasCreate:    {scope: ScopeWorkspace, roles: o | a},
	CanvasUpdate:    {scope: ScopeWorkspace, roles: o | a},
	CanvasDelete:    {scope: ScopeWorkspace, roles: o | a},     // rule 6
	CanvasEdit:      {scope: ScopeWorkspace, roles: o | a | e}, // E: granted; rule 8 (locked)
	CanvasLock:      {scope: ScopeWorkspace, roles: o | a | e},
	CanvasUnlock:    {scope: ScopeWorkspace, roles: o | a | e},
	ShareLinkList:   {scope: ScopeWorkspace, roles: o | a},
	ShareLinkCreate: {scope: ScopeWorkspace, roles: o | a},
	ShareLinkUpdate: {scope: ScopeWorkspace, roles: o | a},
	ShareLinkRevoke: {scope: ScopeWorkspace, roles: o | a},
	ShareInfo:       {scope: ScopeAnonymous}, // token in the path is the credential
	ShareUnlock:     {scope: ScopeAnonymous}, // service rule: rate-limited per link and IP
	ShareView:       {scope: ScopeGuest},
	EventsStream:    {scope: ScopeWorkspace, roles: o | a | e | v},
}

// ScopeOf returns the scope of p. It panics on an unknown key: every operation must use a
// key from this file (caught at start-up when routes are registered).
func ScopeOf(p Permission) Scope {
	r, ok := rules[p]
	if !ok {
		panic("authz: unknown permission " + string(p))
	}
	return r.scope
}

// RoleAllows reports whether a workspace role may use p at all (before service rules).
func RoleAllows(role Role, p Permission) bool {
	r, ok := rules[p]
	if !ok || r.scope != ScopeWorkspace {
		return false
	}
	return r.roles&role.bit() != 0
}

// All returns every known key (used by the contract test).
func All() []Permission {
	out := make([]Permission, 0, len(rules))
	for p := range rules {
		out = append(out, p)
	}
	return out
}
