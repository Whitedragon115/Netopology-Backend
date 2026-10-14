# Backend B2 production function index

Generated from the reviewed source on 2026-10-14. Excludes tests and generated dbgen. Declaration inventory only; see [B2_RECOVERY_REVIEW.md](B2_RECOVERY_REVIEW.md) for behavioral evidence and limitations.

Total: 449 production function declarations. API operation matrix: 72 operations.
| Package/file | Function declaration |
| --- | --- |
| [internal/account/handler.go:18](../internal/account/handler.go#L18) | `func NewHandler(svc *Service) *Handler` |
| [internal/account/handler.go:20](../internal/account/handler.go#L20) | `func (h *Handler) Register(api huma.API)` |
| [internal/account/handler.go:56](../internal/account/handler.go#L56) | `func (h *Handler) getMe(ctx context.Context, _ *struct` |
| [internal/account/handler.go:66](../internal/account/handler.go#L66) | `func (h *Handler) updateMe(ctx context.Context, in *updateMeInput) (*getMeOutput, error)` |
| [internal/account/handler.go:76](../internal/account/handler.go#L76) | `func (h *Handler) changeMyPassword(ctx context.Context, in *changeMyPasswordInput) (*struct` |
| [internal/account/handler.go:90](../internal/account/handler.go#L90) | `func (h *Handler) listMyIdentities(ctx context.Context, _ *struct` |
| [internal/account/handler.go:114](../internal/account/handler.go#L114) | `func (h *Handler) linkMyIdentity(ctx context.Context, in *linkMyIdentityInput) (*linkMyIdentityOutput, error)` |
| [internal/account/handler.go:122](../internal/account/handler.go#L122) | `func (h *Handler) unlinkMyIdentity(ctx context.Context, in *unlinkMyIdentityInput) (*struct` |
| [internal/account/handler.go:135](../internal/account/handler.go#L135) | `func (h *Handler) listUsers(ctx context.Context, in *listUsersInput) (*listUsersOutput, error)` |
| [internal/account/handler.go:148](../internal/account/handler.go#L148) | `func (h *Handler) createUser(ctx context.Context, in *createUserInput) (*accountOutput, error)` |
| [internal/account/handler.go:161](../internal/account/handler.go#L161) | `func (h *Handler) lookupUser(ctx context.Context, in *lookupUserInput) (*lookupUserOutput, error)` |
| [internal/account/handler.go:173](../internal/account/handler.go#L173) | `func (h *Handler) getUser(ctx context.Context, in *userIDInput) (*accountOutput, error)` |
| [internal/account/handler.go:186](../internal/account/handler.go#L186) | `func (h *Handler) updateUser(ctx context.Context, in *updateUserInput) (*accountOutput, error)` |
| [internal/account/handler.go:199](../internal/account/handler.go#L199) | `func (h *Handler) resetUserPassword(ctx context.Context, in *resetUserPasswordInput) (*struct` |
| [internal/account/handler.go:208](../internal/account/handler.go#L208) | `func (h *Handler) getInstanceSettings(ctx context.Context, _ *struct` |
| [internal/account/handler.go:218](../internal/account/handler.go#L218) | `func (h *Handler) updateInstanceSettings(ctx context.Context, in *updateInstanceSettingsInput) (*instanceSettingsOutput, error)` |
| [internal/account/model.go:25](../internal/account/model.go#L25) | `func (m *MeUpdate) UnmarshalJSON(data []byte) error` |
| [internal/account/model.go:58](../internal/account/model.go#L58) | `func identityFrom(row dbgen.UserIdentity) Identity` |
| [internal/account/model.go:81](../internal/account/model.go#L81) | `func (a *AccountUpdate) UnmarshalJSON(data []byte) error` |
| [internal/account/oauth.go:14](../internal/account/oauth.go#L14) | `func (s *Service) LinkMyIdentity(ctx context.Context, provider string) (string, http.Cookie, error)` |
| [internal/account/self.go:18](../internal/account/self.go#L18) | `func accountFrom(ctx context.Context, q *dbgen.Queries, user dbgen.User) (apimodel.Account, error)` |
| [internal/account/self.go:27](../internal/account/self.go#L27) | `func (s *Service) UpdateMe(ctx context.Context, in MeUpdate) (apimodel.Account, error)` |
| [internal/account/self.go:71](../internal/account/self.go#L71) | `func (s *Service) ChangeMyPassword(ctx context.Context, in PasswordChange) error` |
| [internal/account/self.go:121](../internal/account/self.go#L121) | `func (s *Service) ListMyIdentities(ctx context.Context) ([]Identity, bool, error)` |
| [internal/account/self.go:141](../internal/account/self.go#L141) | `func (s *Service) UnlinkMyIdentity(ctx context.Context, id uuid.UUID) error` |
| [internal/account/service.go:24](../internal/account/service.go#L24) | `func NewService(db *pg.DB) *Service` |
| [internal/account/service.go:26](../internal/account/service.go#L26) | `func (s *Service) SetOAuthLinkStarter(starter OAuthLinkStarter)` |
| [internal/account/service.go:29](../internal/account/service.go#L29) | `func (s *Service) GetMe(ctx context.Context) (apimodel.Account, error)` |
| [internal/account/settings.go:15](../internal/account/settings.go#L15) | `func readSettings(ctx context.Context, q *dbgen.Queries) (InstanceSettings, error)` |
| [internal/account/settings.go:38](../internal/account/settings.go#L38) | `func (s *Service) GetInstanceSettings(ctx context.Context) (InstanceSettings, error)` |
| [internal/account/settings.go:45](../internal/account/settings.go#L45) | `func (s *Service) UpdateInstanceSettings(ctx context.Context, in InstanceSettings) (InstanceSettings, error)` |
| [internal/account/users.go:19](../internal/account/users.go#L19) | `func (s *Service) ListUsers(ctx context.Context, page httpx.PageParams, search string) (AccountPage, error)` |
| [internal/account/users.go:49](../internal/account/users.go#L49) | `func (s *Service) CreateUser(ctx context.Context, in AccountCreate) (apimodel.Account, error)` |
| [internal/account/users.go:98](../internal/account/users.go#L98) | `func (s *Service) LookupUser(ctx context.Context, username string) (User, error)` |
| [internal/account/users.go:120](../internal/account/users.go#L120) | `func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (apimodel.Account, error)` |
| [internal/account/users.go:134](../internal/account/users.go#L134) | `func (s *Service) UpdateUser(ctx context.Context, id uuid.UUID, in AccountUpdate) (apimodel.Account, error)` |
| [internal/account/users.go:239](../internal/account/users.go#L239) | `func (s *Service) ResetUserPassword(ctx context.Context, id uuid.UUID, in PasswordReset) error` |
| [internal/apimodel/account.go:37](../internal/apimodel/account.go#L37) | `func AccountFrom(u dbgen.User, providers []string) Account` |
| [internal/audit/handler.go:16](../internal/audit/handler.go#L16) | `func NewHandler(svc *Service) *Handler` |
| [internal/audit/handler.go:17](../internal/audit/handler.go#L17) | `func (h *Handler) Register(api huma.API)` |
| [internal/audit/handler.go:32](../internal/audit/handler.go#L32) | `func (h *Handler) list(ctx context.Context, in *listInput) (*listOutput, error)` |
| [internal/audit/service.go:23](../internal/audit/service.go#L23) | `func NewService(db *pg.DB, az *authz.Resolver) *Service` |
| [internal/audit/service.go:57](../internal/audit/service.go#L57) | `func (s *Service) List(ctx context.Context, in listInput) (AuditPage, error)` |
| [internal/auth/handler.go:16](../internal/auth/handler.go#L16) | `func NewHandler(svc *Service) *Handler` |
| [internal/auth/handler.go:18](../internal/auth/handler.go#L18) | `func (h *Handler) Register(api huma.API)` |
| [internal/auth/handler.go:44](../internal/auth/handler.go#L44) | `func (h *Handler) login(ctx context.Context, in *loginInput) (*loginOutput, error)` |
| [internal/auth/handler.go:54](../internal/auth/handler.go#L54) | `func (h *Handler) providers(ctx context.Context, _ *struct` |
| [internal/auth/handler.go:62](../internal/auth/handler.go#L62) | `func (h *Handler) logout(ctx context.Context, _ *struct` |
| [internal/auth/handler.go:79](../internal/auth/handler.go#L79) | `func (h *Handler) startOAuth(ctx context.Context, in *startOAuthInput) (*startOAuthOutput, error)` |
| [internal/auth/handler.go:98](../internal/auth/handler.go#L98) | `func (h *Handler) finishOAuth(ctx context.Context, in *finishOAuthInput) (*finishOAuthOutput, error)` |
| [internal/auth/middleware.go:23](../internal/auth/middleware.go#L23) | `func Middleware(db *pg.DB, ttl time.Duration, cookies httpx.Cookies, log *slog.Logger) func(http.Handler) http.Handler` |
| [internal/auth/oauth.go:41](../internal/auth/oauth.go#L41) | `func oauthCookieFrom(ctx context.Context) string` |
| [internal/auth/oauth.go:46](../internal/auth/oauth.go#L46) | `func safeReturnTo(value string) string` |
| [internal/auth/oauth.go:61](../internal/auth/oauth.go#L61) | `func (s *Service) oauthRedirect(provider string) string` |
| [internal/auth/oauth.go:66](../internal/auth/oauth.go#L66) | `func (s *Service) StartSignIn(_ context.Context, provider, returnTo string) (string, http.Cookie, error)` |
| [internal/auth/oauth.go:71](../internal/auth/oauth.go#L71) | `func (s *Service) StartLink(_ context.Context, provider string, userID uuid.UUID) (string, http.Cookie, error)` |
| [internal/auth/oauth.go:75](../internal/auth/oauth.go#L75) | `func (s *Service) beginOAuth(provider, mode string, userID uuid.UUID, returnTo string) (string, http.Cookie, error)` |
| [internal/auth/oauth.go:96](../internal/auth/oauth.go#L96) | `func (s *Service) finishFailure(code string, cookies []http.Cookie) OAuthResult` |
| [internal/auth/oauth.go:101](../internal/auth/oauth.go#L101) | `func (s *Service) FinishOAuth(ctx context.Context, provider, code, state, providerError, signedCookie string) OAuthResult` |
| [internal/auth/oauth.go:147](../internal/auth/oauth.go#L147) | `func oauthErrorCode(err error) string` |
| [internal/auth/oauth.go:161](../internal/auth/oauth.go#L161) | `func (s *Service) finishSignIn(ctx context.Context, provider string, profile OAuthProfile) (http.Cookie, error)` |
| [internal/auth/oauth.go:228](../internal/auth/oauth.go#L228) | `func (s *Service) finishLink(ctx context.Context, provider string, userID uuid.UUID, profile OAuthProfile) error` |
| [internal/auth/oauth_provider.go:37](../internal/auth/oauth_provider.go#L37) | `func defaultOAuthProviders(cfg *config.Config) map[string]OAuthProvider` |
| [internal/auth/oauth_provider.go:67](../internal/auth/oauth_provider.go#L67) | `func (p *remoteProvider) Authorize(state, verifier, redirectURI string) string` |
| [internal/auth/oauth_provider.go:73](../internal/auth/oauth_provider.go#L73) | `func (p *remoteProvider) Profile(ctx context.Context, code, verifier, redirectURI string) (OAuthProfile, error)` |
| [internal/auth/ratelimit.go:25](../internal/auth/ratelimit.go#L25) | `func newLoginLimiter(now func() time.Time) *loginLimiter` |
| [internal/auth/ratelimit.go:29](../internal/auth/ratelimit.go#L29) | `func (l *loginLimiter) allow(ip, username string) int` |
| [internal/auth/ratelimit.go:60](../internal/auth/ratelimit.go#L60) | `func refill(b bucket, now time.Time, capacity float64, period time.Duration) (bucket, int)` |
| [internal/auth/service.go:31](../internal/auth/service.go#L31) | `func NewService(db *pg.DB, cfg *config.Config, sessions *Sessions) *Service` |
| [internal/auth/service.go:39](../internal/auth/service.go#L39) | `func (s *Service) SetOAuthProviders(providers map[string]OAuthProvider)` |
| [internal/auth/service.go:42](../internal/auth/service.go#L42) | `func (s *Service) Providers(_ context.Context) AuthProviders` |
| [internal/auth/service.go:54](../internal/auth/service.go#L54) | `func (s *Service) Login(ctx context.Context, in LoginRequest) (apimodel.Account, http.Cookie, error)` |
| [internal/auth/service.go:142](../internal/auth/service.go#L142) | `func (s *Service) invalidLogin(ctx context.Context, userID *uuid.UUID) (apimodel.Account, http.Cookie, error)` |
| [internal/auth/service.go:149](../internal/auth/service.go#L149) | `func (s *Service) failedLogin(ctx context.Context, userID *uuid.UUID, reason string) error` |
| [internal/auth/service.go:159](../internal/auth/service.go#L159) | `func (s *Service) Logout(ctx context.Context) (http.Cookie, error)` |
| [internal/auth/session.go:27](../internal/auth/session.go#L27) | `func sessionLifetime(ttl time.Duration, remember bool) time.Duration` |
| [internal/auth/session.go:36](../internal/auth/session.go#L36) | `func (s *Sessions) Start(ctx context.Context, q *dbgen.Queries, userID uuid.UUID, remember bool) (http.Cookie, error)` |
| [internal/authz/actor.go:24](../internal/authz/actor.go#L24) | `func (r Role) bit() roleSet` |
| [internal/authz/actor.go:64](../internal/authz/actor.go#L64) | `func (p *Principal) SignedIn() bool` |
| [internal/authz/actor.go:67](../internal/authz/actor.go#L67) | `func (p *Principal) IsInstanceStaff() bool` |
| [internal/authz/actor.go:73](../internal/authz/actor.go#L73) | `func WithPrincipal(ctx context.Context, p *Principal) context.Context` |
| [internal/authz/actor.go:78](../internal/authz/actor.go#L78) | `func PrincipalFrom(ctx context.Context) *Principal` |
| [internal/authz/actor.go:86](../internal/authz/actor.go#L86) | `func RequireSignedIn(ctx context.Context) (*Principal, error)` |
| [internal/authz/actor.go:95](../internal/authz/actor.go#L95) | `func RequireInstanceStaff(ctx context.Context) (*Principal, error)` |
| [internal/authz/actor.go:125](../internal/authz/actor.go#L125) | `func (a *Actor) UserID() uuid.UUID` |
| [internal/authz/actor.go:128](../internal/authz/actor.go#L128) | `func (a *Actor) IsManager() bool` |
| [internal/authz/actor.go:131](../internal/authz/actor.go#L131) | `func (a *Actor) SeesSensitive() bool` |
| [internal/authz/actor.go:134](../internal/authz/actor.go#L134) | `func (a *Actor) Can(p Permission) bool` |
| [internal/authz/actor.go:137](../internal/authz/actor.go#L137) | `func (a *Actor) Require(p Permission) error` |
| [internal/authz/actor.go:147](../internal/authz/actor.go#L147) | `func (a *Actor) CanvasGranted(ctx context.Context, canvasID uuid.UUID) (bool, error)` |
| [internal/authz/actor.go:159](../internal/authz/actor.go#L159) | `func (a *Actor) GrantedCanvasIDs(ctx context.Context) (map[uuid.UUID]bool, error)` |
| [internal/authz/actor.go:182](../internal/authz/actor.go#L182) | `func NewResolver(db *pg.DB) *Resolver` |
| [internal/authz/actor.go:185](../internal/authz/actor.go#L185) | `func (r *Resolver) Actor(ctx context.Context, workspaceID uuid.UUID, perm Permission) (*Actor, error)` |
| [internal/authz/permission.go:214](../internal/authz/permission.go#L214) | `func ScopeOf(p Permission) Scope` |
| [internal/authz/permission.go:223](../internal/authz/permission.go#L223) | `func RoleAllows(role Role, p Permission) bool` |
| [internal/authz/permission.go:232](../internal/authz/permission.go#L232) | `func All() []Permission` |
| [internal/canvas/feature.go:46](../internal/canvas/feature.go#L46) | `func NewService(db *pg.DB, az *authz.Resolver, reader InventoryReader) *Service` |
| [internal/canvas/feature.go:55](../internal/canvas/feature.go#L55) | `func NewHandler(svc *Service) *Handler` |
| [internal/canvas/feature.go:59](../internal/canvas/feature.go#L59) | `func (h *Handler) Register(api huma.API)` |
| [internal/canvas/guest.go:16](../internal/canvas/guest.go#L16) | `func (s *Service) ReadForGuest(ctx context.Context, q *dbgen.Queries) (apimodel.CanvasView, error)` |
| [internal/canvas/handler.go:31](../internal/canvas/handler.go#L31) | `func (h *Handler) list(ctx context.Context, in *workspaceInput) (*listOutput, error)` |
| [internal/canvas/handler.go:39](../internal/canvas/handler.go#L39) | `func (h *Handler) get(ctx context.Context, in *canvasInput) (*canvasOutput, error)` |
| [internal/canvas/handler.go:47](../internal/canvas/handler.go#L47) | `func (h *Handler) getView(ctx context.Context, in *canvasInput) (*viewOutput, error)` |
| [internal/canvas/handler.go:55](../internal/canvas/handler.go#L55) | `func (h *Handler) getAreaView(ctx context.Context, in *areaInput) (*viewOutput, error)` |
| [internal/canvas/handler.go:81](../internal/canvas/handler.go#L81) | `func (h *Handler) create(ctx context.Context, in *createInput) (*canvasOutput, error)` |
| [internal/canvas/handler.go:89](../internal/canvas/handler.go#L89) | `func (h *Handler) update(ctx context.Context, in *updateInput) (*canvasOutput, error)` |
| [internal/canvas/handler.go:105](../internal/canvas/handler.go#L105) | `func (h *Handler) delete(ctx context.Context, in *deleteInput) (*struct` |
| [internal/canvas/lock.go:15](../internal/canvas/lock.go#L15) | `func (s *Service) SetLock(ctx context.Context, wsID, id uuid.UUID, locked bool) (Canvas, error)` |
| [internal/canvas/operations.go:18](../internal/canvas/operations.go#L18) | `func marshalObject(value map[string]any) (json.RawMessage, error)` |
| [internal/canvas/operations.go:25](../internal/canvas/operations.go#L25) | `func number(value *float64) float64` |
| [internal/canvas/operations.go:32](../internal/canvas/operations.go#L32) | `func integer(value *int32) int32` |
| [internal/canvas/operations.go:39](../internal/canvas/operations.go#L39) | `func boolean(value *bool) bool` |
| [internal/canvas/operations.go:46](../internal/canvas/operations.go#L46) | `func operationError(index int, err error) error` |
| [internal/canvas/operations.go:67](../internal/canvas/operations.go#L67) | `func (s *Service) ApplyOperations(ctx context.Context, wsID, canvasID uuid.UUID, batch CanvasOperationBatch) (CanvasOperationResult, error)` |
| [internal/canvas/operations.go:141](../internal/canvas/operations.go#L141) | `func (s *Service) applyOne(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, canvasID uuid.UUID,` |
| [internal/canvas/operations.go:299](../internal/canvas/operations.go#L299) | `func (s *Service) createShape(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, canvasID uuid.UUID, result CanvasOperationResultItem, op CanvasOperation) (CanvasOperationResultItem, error)` |
| [internal/canvas/operations.go:323](../internal/canvas/operations.go#L323) | `func (s *Service) updateShape(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, canvasID uuid.UUID, result CanvasOperationResultItem, op CanvasOperation) (CanvasOperationResultItem, error)` |
| [internal/canvas/operations.go:375](../internal/canvas/operations.go#L375) | `func (s *Service) overrideLink(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, canvasID uuid.UUID, result CanvasOperationResultItem, op CanvasOperation) (CanvasOperationResultItem, error)` |
| [internal/canvas/operations_handler.go:25](../internal/canvas/operations_handler.go#L25) | `func (h *Handler) applyOperations(ctx context.Context, in *operationsInput) (*operationsOutput, error)` |
| [internal/canvas/operations_handler.go:33](../internal/canvas/operations_handler.go#L33) | `func (h *Handler) lock(ctx context.Context, in *canvasInput) (*lockOutput, error)` |
| [internal/canvas/operations_handler.go:41](../internal/canvas/operations_handler.go#L41) | `func (h *Handler) unlock(ctx context.Context, in *canvasInput) (*lockOutput, error)` |
| [internal/canvas/operations_handler.go:49](../internal/canvas/operations_handler.go#L49) | `func (h *Handler) registerOperations(api huma.API)` |
| [internal/canvas/operations_model.go:37](../internal/canvas/operations_model.go#L37) | `func (v *CanvasOperation) UnmarshalJSON(data []byte) error` |
| [internal/canvas/operations_schema.go:11](../internal/canvas/operations_schema.go#L11) | `func (CanvasOperation) TransformSchema(r huma.Registry, flat *huma.Schema) *huma.Schema` |
| [internal/canvas/read.go:17](../internal/canvas/read.go#L17) | `func fromRow(row dbgen.Canvase) (Canvas, error)` |
| [internal/canvas/read.go:39](../internal/canvas/read.go#L39) | `func (s *Service) canSeeCanvas(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, row dbgen.Canvase) error` |
| [internal/canvas/read.go:55](../internal/canvas/read.go#L55) | `func (s *Service) List(ctx context.Context, wsID uuid.UUID) (CanvasPage, error)` |
| [internal/canvas/read.go:82](../internal/canvas/read.go#L82) | `func (s *Service) Get(ctx context.Context, wsID, id uuid.UUID) (Canvas, error)` |
| [internal/canvas/read.go:105](../internal/canvas/read.go#L105) | `func (s *Service) GetView(ctx context.Context, wsID, id uuid.UUID) (CanvasView, error)` |
| [internal/canvas/read.go:128](../internal/canvas/read.go#L128) | `func (s *Service) view(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, row dbgen.Canvase) (CanvasView, error)` |
| [internal/canvas/read.go:274](../internal/canvas/read.go#L274) | `func actorID(actor *authz.Actor) *uuid.UUID` |
| [internal/canvas/read.go:276](../internal/canvas/read.go#L276) | `func placementFromRow(p dbgen.Placement) (Placement, error)` |
| [internal/canvas/read.go:296](../internal/canvas/read.go#L296) | `func overrideFromRow(p dbgen.Placement) (LinkOverride, error)` |
| [internal/canvas/read.go:309](../internal/canvas/read.go#L309) | `func shapeFromRow(p dbgen.CanvasShape) (Shape, error)` |
| [internal/canvas/write.go:18](../internal/canvas/write.go#L18) | `func canvasChange(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, action authz.Permission, event events.Type, id uuid.UUID) error` |
| [internal/canvas/write.go:25](../internal/canvas/write.go#L25) | `func canvasError(err error) error` |
| [internal/canvas/write.go:35](../internal/canvas/write.go#L35) | `func (s *Service) CanvasCreate(ctx context.Context, wsID uuid.UUID, req CanvasCreate) (Canvas, error)` |
| [internal/canvas/write.go:69](../internal/canvas/write.go#L69) | `func (s *Service) CanvasUpdate(ctx context.Context, wsID, id uuid.UUID, req CanvasUpdate, match *int32) (Canvas, error)` |
| [internal/canvas/write.go:139](../internal/canvas/write.go#L139) | `func (s *Service) Delete(ctx context.Context, wsID, id uuid.UUID, match *int32) error` |
| [internal/canvas/write.go:170](../internal/canvas/write.go#L170) | `func (s *Service) GetAreaView(ctx context.Context, wsID, areaID uuid.UUID) (CanvasView, error)` |
| [internal/events/handler.go:20](../internal/events/handler.go#L20) | `func NewHandler(svc *Service) *Handler` |
| [internal/events/handler.go:21](../internal/events/handler.go#L21) | `func (h *Handler) Register(api huma.API)` |
| [internal/events/handler.go:37](../internal/events/handler.go#L37) | `func (h *Handler) stream(ctx context.Context, in *streamInput) (*huma.StreamResponse, error)` |
| [internal/events/service.go:36](../internal/events/service.go#L36) | `func NewService(db *pg.DB, az *authz.Resolver, broker *platformevents.Broker, inventory InventoryReader, canvas ResourceReader, member MemberReader) *Service` |
| [internal/events/service.go:57](../internal/events/service.go#L57) | `func (s *Service) Open(ctx context.Context, wsID uuid.UUID, canvasID string, lastID string) (*Stream, error)` |
| [internal/events/service.go:82](../internal/events/service.go#L82) | `func (s *Stream) Close()` |
| [internal/events/service.go:84](../internal/events/service.go#L84) | `func (s *Stream) Run(ctx context.Context, send func(*Event) error) error` |
| [internal/events/service.go:170](../internal/events/service.go#L170) | `func (s *Service) project(ctx context.Context, row dbgen.WorkspaceEvent, canvasID *uuid.UUID) (*Event, error)` |
| [internal/inventory/convert.go:14](../internal/inventory/convert.go#L14) | `func areaBase(id, wsID uuid.UUID, parentID *uuid.UUID, name, kind, color string,` |
| [internal/inventory/convert.go:25](../internal/inventory/convert.go#L25) | `func properties(raw json.RawMessage) ([]Field, error)` |
| [internal/inventory/convert.go:33](../internal/inventory/convert.go#L33) | `func areaFromRow(row dbgen.Area, nodeCount int64, audience Audience) (Area, error)` |
| [internal/inventory/convert.go:52](../internal/inventory/convert.go#L52) | `func nodeFromRow(row dbgen.Node, audience Audience) (Node, error)` |
| [internal/inventory/convert.go:68](../internal/inventory/convert.go#L68) | `func linkFromRow(row dbgen.Link, audience Audience) (Link, error)` |
| [internal/inventory/convert.go:84](../internal/inventory/convert.go#L84) | `func noteFromRow(row dbgen.Note, audience Audience) (Note, bool)` |
| [internal/inventory/feature.go:51](../internal/inventory/feature.go#L51) | `func NewService(db *pg.DB, az *authz.Resolver) *Service` |
| [internal/inventory/feature.go:60](../internal/inventory/feature.go#L60) | `func NewHandler(svc *Service) *Handler` |
| [internal/inventory/feature.go:63](../internal/inventory/feature.go#L63) | `func (h *Handler) Register(api huma.API)` |
| [internal/inventory/feature.go:92](../internal/inventory/feature.go#L92) | `func (h *Handler) listAreas(ctx context.Context, in *listAreasInput) (*areaPageOutput, error)` |
| [internal/inventory/feature.go:111](../internal/inventory/feature.go#L111) | `func (h *Handler) getArea(ctx context.Context, in *getAreaInput) (*areaOutput, error)` |
| [internal/inventory/feature.go:130](../internal/inventory/feature.go#L130) | `func (h *Handler) listNodes(ctx context.Context, in *listNodesInput) (*nodePageOutput, error)` |
| [internal/inventory/feature.go:160](../internal/inventory/feature.go#L160) | `func (h *Handler) getNode(ctx context.Context, in *getNodeInput) (*nodeOutput, error)` |
| [internal/inventory/feature.go:170](../internal/inventory/feature.go#L170) | `func (h *Handler) getNodeUsage(ctx context.Context, in *getNodeInput) (*nodeUsageOutput, error)` |
| [internal/inventory/feature.go:186](../internal/inventory/feature.go#L186) | `func (h *Handler) listLinks(ctx context.Context, in *listLinksInput) (*linkPageOutput, error)` |
| [internal/inventory/feature.go:209](../internal/inventory/feature.go#L209) | `func (h *Handler) getLink(ctx context.Context, in *getLinkInput) (*linkOutput, error)` |
| [internal/inventory/feature.go:228](../internal/inventory/feature.go#L228) | `func (h *Handler) listNotes(ctx context.Context, in *listNotesInput) (*notePageOutput, error)` |
| [internal/inventory/feature.go:255](../internal/inventory/feature.go#L255) | `func optionalID(raw string) (*uuid.UUID, error)` |
| [internal/inventory/feature.go:277](../internal/inventory/feature.go#L277) | `func (h *Handler) getNote(ctx context.Context, in *getNoteInput) (*noteOutput, error)` |
| [internal/inventory/read.go:23](../internal/inventory/read.go#L23) | `func visible(ctx context.Context, q *dbgen.Queries, actor *authz.Actor) (visibility, error)` |
| [internal/inventory/read.go:34](../internal/inventory/read.go#L34) | `func audience(actor *authz.Actor) Audience` |
| [internal/inventory/read.go:36](../internal/inventory/read.go#L36) | `func pageCursor(cursor string) (*uuid.UUID, error)` |
| [internal/inventory/read.go:51](../internal/inventory/read.go#L51) | `func pageLimit(limit int) int32` |
| [internal/inventory/read.go:61](../internal/inventory/read.go#L61) | `func more[T any](rows []T, limit int, id func(T) uuid.UUID) ([]T, *string)` |
| [internal/inventory/read.go:98](../internal/inventory/read.go#L98) | `func (s *Service) ListAreas(ctx context.Context, wsID uuid.UUID, in AreaFilter) (AreaPage, error)` |
| [internal/inventory/read.go:142](../internal/inventory/read.go#L142) | `func (s *Service) GetArea(ctx context.Context, wsID, id uuid.UUID) (Area, error)` |
| [internal/inventory/read.go:166](../internal/inventory/read.go#L166) | `func (s *Service) ListNodes(ctx context.Context, wsID uuid.UUID, in NodeFilter) (NodePage, error)` |
| [internal/inventory/read.go:202](../internal/inventory/read.go#L202) | `func (s *Service) GetNode(ctx context.Context, wsID, id uuid.UUID) (Node, error)` |
| [internal/inventory/read.go:226](../internal/inventory/read.go#L226) | `func (s *Service) GetNodeUsage(ctx context.Context, wsID, id uuid.UUID) (NodeUsage, error)` |
| [internal/inventory/read.go:267](../internal/inventory/read.go#L267) | `func (s *Service) ListLinks(ctx context.Context, wsID uuid.UUID, in LinkFilter) (LinkPage, error)` |
| [internal/inventory/read.go:302](../internal/inventory/read.go#L302) | `func (s *Service) GetLink(ctx context.Context, wsID, id uuid.UUID) (Link, error)` |
| [internal/inventory/read.go:326](../internal/inventory/read.go#L326) | `func (s *Service) ListNotes(ctx context.Context, wsID uuid.UUID, in NoteFilter) (NotePage, error)` |
| [internal/inventory/read.go:365](../internal/inventory/read.go#L365) | `func (s *Service) GetNote(ctx context.Context, wsID, id uuid.UUID) (Note, error)` |
| [internal/inventory/read.go:393](../internal/inventory/read.go#L393) | `func noteFilterTarget(in NoteFilter) (string, *uuid.UUID, error)` |
| [internal/inventory/read.go:414](../internal/inventory/read.go#L414) | `func cleanSearch(query string) *string` |
| [internal/inventory/redact.go:6](../internal/inventory/redact.go#L6) | `func redactFields(fields []Field, audience Audience) []Field` |
| [internal/inventory/redact.go:21](../internal/inventory/redact.go#L21) | `func redactNode(node *Node, audience Audience)` |
| [internal/inventory/redact.go:29](../internal/inventory/redact.go#L29) | `func redactLink(link *Link, audience Audience)` |
| [internal/inventory/redact.go:36](../internal/inventory/redact.go#L36) | `func redactArea(area *Area, audience Audience)` |
| [internal/inventory/redact.go:44](../internal/inventory/redact.go#L44) | `func redactNote(note *Note, audience Audience) bool` |
| [internal/inventory/snapshot.go:16](../internal/inventory/snapshot.go#L16) | `func (s *Service) ReadForActor(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, ids IDs) (Snapshot, error)` |
| [internal/inventory/snapshot.go:29](../internal/inventory/snapshot.go#L29) | `func (s *Service) ReadForGuest(ctx context.Context, q *dbgen.Queries, ids IDs) (Snapshot, error)` |
| [internal/inventory/snapshot.go:37](../internal/inventory/snapshot.go#L37) | `func readSnapshot(ctx context.Context, q *dbgen.Queries, wsID uuid.UUID, vis visibility, audience Audience, ids IDs) (Snapshot, error)` |
| [internal/inventory/write_area.go:16](../internal/inventory/write_area.go#L16) | `func (s *Service) CreateArea(ctx context.Context, wsID uuid.UUID, req AreaCreate) (Area, error)` |
| [internal/inventory/write_area.go:74](../internal/inventory/write_area.go#L74) | `func (s *Service) UpdateArea(ctx context.Context, wsID, id uuid.UUID, req AreaUpdate, match *int32) (Area, error)` |
| [internal/inventory/write_area.go:154](../internal/inventory/write_area.go#L154) | `func (s *Service) DeleteArea(ctx context.Context, wsID, id uuid.UUID, match *int32) error` |
| [internal/inventory/write_area.go:196](../internal/inventory/write_area.go#L196) | `func auditActor(actor *authz.Actor) *uuid.UUID` |
| [internal/inventory/write_handler.go:14](../internal/inventory/write_handler.go#L14) | `func (h *Handler) registerWrites(api huma.API)` |
| [internal/inventory/write_handler.go:43](../internal/inventory/write_handler.go#L43) | `func matchHeader(header string) (*int32, error)` |
| [internal/inventory/write_handler.go:73](../internal/inventory/write_handler.go#L73) | `func (h *Handler) createArea(ctx context.Context, in *areaCreateInput) (*areaWriteOutput, error)` |
| [internal/inventory/write_handler.go:81](../internal/inventory/write_handler.go#L81) | `func (h *Handler) updateArea(ctx context.Context, in *areaUpdateInput) (*areaWriteOutput, error)` |
| [internal/inventory/write_handler.go:93](../internal/inventory/write_handler.go#L93) | `func (h *Handler) deleteArea(ctx context.Context, in *areaDeleteInput) (*struct` |
| [internal/inventory/write_handler.go:132](../internal/inventory/write_handler.go#L132) | `func (h *Handler) createNode(ctx context.Context, in *nodeCreateInput) (*nodeWriteOutput, error)` |
| [internal/inventory/write_handler.go:140](../internal/inventory/write_handler.go#L140) | `func (h *Handler) updateNode(ctx context.Context, in *nodeUpdateInput) (*nodeWriteOutput, error)` |
| [internal/inventory/write_handler.go:152](../internal/inventory/write_handler.go#L152) | `func (h *Handler) deleteNode(ctx context.Context, in *nodeDeleteInput) (*struct` |
| [internal/inventory/write_handler.go:163](../internal/inventory/write_handler.go#L163) | `func (h *Handler) setNodeHealth(ctx context.Context, in *nodeHealthInput) (*nodeHealthOutput, error)` |
| [internal/inventory/write_handler.go:193](../internal/inventory/write_handler.go#L193) | `func (h *Handler) createLink(ctx context.Context, in *linkCreateInput) (*linkWriteOutput, error)` |
| [internal/inventory/write_handler.go:201](../internal/inventory/write_handler.go#L201) | `func (h *Handler) updateLink(ctx context.Context, in *linkUpdateInput) (*linkWriteOutput, error)` |
| [internal/inventory/write_handler.go:213](../internal/inventory/write_handler.go#L213) | `func (h *Handler) deleteLink(ctx context.Context, in *linkDeleteInput) (*struct` |
| [internal/inventory/write_handler.go:246](../internal/inventory/write_handler.go#L246) | `func (h *Handler) createNote(ctx context.Context, in *noteCreateInput) (*noteWriteOutput, error)` |
| [internal/inventory/write_handler.go:254](../internal/inventory/write_handler.go#L254) | `func (h *Handler) updateNote(ctx context.Context, in *noteUpdateInput) (*noteWriteOutput, error)` |
| [internal/inventory/write_handler.go:266](../internal/inventory/write_handler.go#L266) | `func (h *Handler) deleteNote(ctx context.Context, in *noteDeleteInput) (*struct` |
| [internal/inventory/write_helpers.go:23](../internal/inventory/write_helpers.go#L23) | `func checkedFields(in []Field, old json.RawMessage) (json.RawMessage, error)` |
| [internal/inventory/write_helpers.go:56](../internal/inventory/write_helpers.go#L56) | `func parsedCIDR(raw *string) (*netip.Prefix, error)` |
| [internal/inventory/write_helpers.go:67](../internal/inventory/write_helpers.go#L67) | `func choose(value, fallback string) string` |
| [internal/inventory/write_helpers.go:74](../internal/inventory/write_helpers.go#L74) | `func idIn(ids []uuid.UUID, id uuid.UUID) bool` |
| [internal/inventory/write_helpers.go:83](../internal/inventory/write_helpers.go#L83) | `func visibleItem(vis visibility, kind string, id uuid.UUID) bool` |
| [internal/inventory/write_helpers.go:102](../internal/inventory/write_helpers.go#L102) | `func requireVisible(vis visibility, kind string, id uuid.UUID) error` |
| [internal/inventory/write_helpers.go:109](../internal/inventory/write_helpers.go#L109) | `func checkMatch(version int32, match *int32) error` |
| [internal/inventory/write_helpers.go:116](../internal/inventory/write_helpers.go#L116) | `func (s *Service) preparePlaceOn(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, place *PlaceOn) error` |
| [internal/inventory/write_helpers.go:148](../internal/inventory/write_helpers.go#L148) | `func insertPlaceOn(ctx context.Context, q *dbgen.Queries, wsID uuid.UUID, place *PlaceOn, kind string, id uuid.UUID) error` |
| [internal/inventory/write_helpers.go:175](../internal/inventory/write_helpers.go#L175) | `func usedElsewhere(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, kind string, id uuid.UUID) error` |
| [internal/inventory/write_helpers.go:189](../internal/inventory/write_helpers.go#L189) | `func recordInventoryChange(ctx context.Context, q *dbgen.Queries, actor *authz.Actor, permission authz.Permission, kind string, event events.Type, id uuid.UUID, canvasIDs []uuid.UUID) error` |
| [internal/inventory/write_helpers.go:203](../internal/inventory/write_helpers.go#L203) | `func inventoryWriteError(err error) error` |
| [internal/inventory/write_link_note.go:15](../internal/inventory/write_link_note.go#L15) | `func (s *Service) CreateLink(ctx context.Context, wsID uuid.UUID, req LinkCreate) (Link, error)` |
| [internal/inventory/write_link_note.go:65](../internal/inventory/write_link_note.go#L65) | `func (s *Service) UpdateLink(ctx context.Context, wsID, id uuid.UUID, req LinkUpdate, match *int32) (Link, error)` |
| [internal/inventory/write_link_note.go:128](../internal/inventory/write_link_note.go#L128) | `func (s *Service) DeleteLink(ctx context.Context, wsID, id uuid.UUID, match *int32) error` |
| [internal/inventory/write_link_note.go:167](../internal/inventory/write_link_note.go#L167) | `func noteTargetIDs(target NoteTarget) (node, link, area, canvas *uuid.UUID, error error)` |
| [internal/inventory/write_link_note.go:190](../internal/inventory/write_link_note.go#L190) | `func (s *Service) CreateNote(ctx context.Context, wsID uuid.UUID, req NoteCreate) (Note, error)` |
| [internal/inventory/write_link_note.go:236](../internal/inventory/write_link_note.go#L236) | `func (s *Service) UpdateNote(ctx context.Context, wsID, id uuid.UUID, req NoteUpdate, match *int32) (Note, error)` |
| [internal/inventory/write_link_note.go:295](../internal/inventory/write_link_note.go#L295) | `func (s *Service) DeleteNote(ctx context.Context, wsID, id uuid.UUID, match *int32) error` |
| [internal/inventory/write_model.go:9](../internal/inventory/write_model.go#L9) | `func presentFields(data []byte) (map[string]json.RawMessage, error)` |
| [internal/inventory/write_model.go:49](../internal/inventory/write_model.go#L49) | `func (v *AreaUpdate) UnmarshalJSON(data []byte) error` |
| [internal/inventory/write_model.go:66](../internal/inventory/write_model.go#L66) | `func (v AreaUpdate) hasChange() bool` |
| [internal/inventory/write_model.go:96](../internal/inventory/write_model.go#L96) | `func (v *NodeUpdate) UnmarshalJSON(data []byte) error` |
| [internal/inventory/write_model.go:112](../internal/inventory/write_model.go#L112) | `func (v NodeUpdate) hasChange() bool` |
| [internal/inventory/write_model.go:142](../internal/inventory/write_model.go#L142) | `func (v *LinkUpdate) UnmarshalJSON(data []byte) error` |
| [internal/inventory/write_model.go:158](../internal/inventory/write_model.go#L158) | `func (v LinkUpdate) hasChange() bool` |
| [internal/inventory/write_model.go:180](../internal/inventory/write_model.go#L180) | `func (v NoteUpdate) hasChange() bool` |
| [internal/inventory/write_node.go:18](../internal/inventory/write_node.go#L18) | `func generatedNodeSlug(ctx context.Context, q *dbgen.Queries, wsID uuid.UUID, name string) (string, error)` |
| [internal/inventory/write_node.go:37](../internal/inventory/write_node.go#L37) | `func (s *Service) CreateNode(ctx context.Context, wsID uuid.UUID, req NodeCreate) (Node, error)` |
| [internal/inventory/write_node.go:103](../internal/inventory/write_node.go#L103) | `func (s *Service) UpdateNode(ctx context.Context, wsID, id uuid.UUID, req NodeUpdate, match *int32) (Node, error)` |
| [internal/inventory/write_node.go:186](../internal/inventory/write_node.go#L186) | `func (s *Service) DeleteNode(ctx context.Context, wsID, id uuid.UUID, match *int32) error` |
| [internal/inventory/write_node.go:231](../internal/inventory/write_node.go#L231) | `func (s *Service) SetNodeHealth(ctx context.Context, wsID, id uuid.UUID, req HealthSet) (Node, error)` |
| [internal/member/feature.go:33](../internal/member/feature.go#L33) | `func NewService(db *pg.DB, az *authz.Resolver) *Service` |
| [internal/member/feature.go:42](../internal/member/feature.go#L42) | `func NewHandler(svc *Service) *Handler` |
| [internal/member/feature.go:45](../internal/member/feature.go#L45) | `func (h *Handler) Register(api huma.API)` |
| [internal/member/feature.go:69](../internal/member/feature.go#L69) | `func (h *Handler) list(ctx context.Context, in *listInput) (*listOutput, error)` |
| [internal/member/feature.go:83](../internal/member/feature.go#L83) | `func (h *Handler) add(ctx context.Context, in *addInput) (*memberOutput, error)` |
| [internal/member/feature.go:97](../internal/member/feature.go#L97) | `func (h *Handler) update(ctx context.Context, in *updateInput) (*memberOutput, error)` |
| [internal/member/feature.go:110](../internal/member/feature.go#L110) | `func (h *Handler) remove(ctx context.Context, in *removeInput) (*struct` |
| [internal/member/feature.go:123](../internal/member/feature.go#L123) | `func (h *Handler) setGrants(ctx context.Context, in *grantsInput) (*memberOutput, error)` |
| [internal/member/service.go:25](../internal/member/service.go#L25) | `func rowFromGet(row dbgen.GetMemberRow) memberRow` |
| [internal/member/service.go:29](../internal/member/service.go#L29) | `func rowFromList(row dbgen.ListMembersRow) memberRow` |
| [internal/member/service.go:33](../internal/member/service.go#L33) | `func render(ctx context.Context, q *dbgen.Queries, wsID uuid.UUID, row memberRow, manager bool) (Member, error)` |
| [internal/member/service.go:53](../internal/member/service.go#L53) | `func (s *Service) List(ctx context.Context, wsID uuid.UUID) (MemberPage, error)` |
| [internal/member/service.go:76](../internal/member/service.go#L76) | `func (s *Service) Add(ctx context.Context, wsID uuid.UUID, in MemberAdd) (Member, error)` |
| [internal/member/service.go:124](../internal/member/service.go#L124) | `func (s *Service) Update(ctx context.Context, wsID, userID uuid.UUID, in MemberUpdate) (Member, error)` |
| [internal/member/service.go:183](../internal/member/service.go#L183) | `func (s *Service) Remove(ctx context.Context, wsID, userID uuid.UUID) error` |
| [internal/member/service.go:217](../internal/member/service.go#L217) | `func (s *Service) SetGrants(ctx context.Context, wsID, userID uuid.UUID, in CanvasGrants) (Member, error)` |
| [internal/member/service.go:252](../internal/member/service.go#L252) | `func (s *Service) Get(ctx context.Context, wsID, userID uuid.UUID) (Member, error)` |
| [internal/member/service.go:272](../internal/member/service.go#L272) | `func load(ctx context.Context, q *dbgen.Queries, wsID, userID uuid.UUID) (Member, error)` |
| [internal/member/service.go:280](../internal/member/service.go#L280) | `func requireAnotherOwner(ctx context.Context, q *dbgen.Queries, wsID uuid.UUID) error` |
| [internal/member/service.go:291](../internal/member/service.go#L291) | `func validRole(role string) bool` |
| [internal/member/service.go:299](../internal/member/service.go#L299) | `func invalidGrants() error` |
| [internal/member/service.go:303](../internal/member/service.go#L303) | `func validateGrants(ctx context.Context, q *dbgen.Queries, wsID uuid.UUID, ids []uuid.UUID) error` |
| [internal/platform/apperr/apperr.go:50](../internal/platform/apperr/apperr.go#L50) | `func (e *Error) Error() string` |
| [internal/platform/apperr/apperr.go:57](../internal/platform/apperr/apperr.go#L57) | `func (e *Error) Unwrap() error` |
| [internal/platform/apperr/apperr.go:60](../internal/platform/apperr/apperr.go#L60) | `func (e *Error) Wrap(cause error) *Error` |
| [internal/platform/apperr/apperr.go:63](../internal/platform/apperr/apperr.go#L63) | `func As(err error) (*Error, bool)` |
| [internal/platform/apperr/apperr.go:70](../internal/platform/apperr/apperr.go#L70) | `func IsKind(err error, k Kind) bool` |
| [internal/platform/apperr/apperr.go:75](../internal/platform/apperr/apperr.go#L75) | `func newErr(k Kind, code, detail string) *Error` |
| [internal/platform/apperr/apperr.go:77](../internal/platform/apperr/apperr.go#L77) | `func Invalid(code, detail string, fields ...FieldError) *Error` |
| [internal/platform/apperr/apperr.go:83](../internal/platform/apperr/apperr.go#L83) | `func Unauthenticated(code, detail string) *Error` |
| [internal/platform/apperr/apperr.go:84](../internal/platform/apperr/apperr.go#L84) | `func Forbidden(code, detail string) *Error` |
| [internal/platform/apperr/apperr.go:85](../internal/platform/apperr/apperr.go#L85) | `func NotFound(code, detail string) *Error` |
| [internal/platform/apperr/apperr.go:86](../internal/platform/apperr/apperr.go#L86) | `func Conflict(code, detail string) *Error` |
| [internal/platform/apperr/apperr.go:87](../internal/platform/apperr/apperr.go#L87) | `func Locked(code, detail string) *Error` |
| [internal/platform/apperr/apperr.go:88](../internal/platform/apperr/apperr.go#L88) | `func PreconditionFailed() *Error` |
| [internal/platform/apperr/apperr.go:92](../internal/platform/apperr/apperr.go#L92) | `func RateLimited(retryAfter int) *Error` |
| [internal/platform/apperr/apperr.go:116](../internal/platform/apperr/apperr.go#L116) | `func ErrNotFound() *Error` |
| [internal/platform/apperr/apperr.go:119](../internal/platform/apperr/apperr.go#L119) | `func ErrRoleInsufficient() *Error` |
| [internal/platform/auditlog/auditlog.go:31](../internal/platform/auditlog/auditlog.go#L31) | `func Write(ctx context.Context, q *dbgen.Queries, e Entry) error` |
| [internal/platform/auditlog/auditlog.go:78](../internal/platform/auditlog/auditlog.go#L78) | `func Ptr(id uuid.UUID) *uuid.UUID` |
| [internal/platform/config/config.go:63](../internal/platform/config/config.go#L63) | `func Load() (*Config, error)` |
| [internal/platform/config/config.go:100](../internal/platform/config/config.go#L100) | `func (c *Config) Dev() bool` |
| [internal/platform/config/config.go:103](../internal/platform/config/config.go#L103) | `func (c *Config) GoogleEnabled() bool` |
| [internal/platform/config/config.go:104](../internal/platform/config/config.go#L104) | `func (c *Config) DiscordEnabled() bool` |
| [internal/platform/crypto/password.go:34](../internal/platform/crypto/password.go#L34) | `func HashPassword(password string) (string, error)` |
| [internal/platform/crypto/password.go:46](../internal/platform/crypto/password.go#L46) | `func VerifyPassword(password, encoded string) (bool, error)` |
| [internal/platform/crypto/password.go:56](../internal/platform/crypto/password.go#L56) | `func NeedsRehash(encoded string) bool` |
| [internal/platform/crypto/password.go:63](../internal/platform/crypto/password.go#L63) | `func DummyVerify(password string)` |
| [internal/platform/crypto/password.go:73](../internal/platform/crypto/password.go#L73) | `func decode(encoded string) (params, []byte, []byte, error)` |
| [internal/platform/crypto/token.go:15](../internal/platform/crypto/token.go#L15) | `func NewToken() (string, error)` |
| [internal/platform/crypto/token.go:24](../internal/platform/crypto/token.go#L24) | `func HashToken(token string) []byte` |
| [internal/platform/crypto/token.go:34](../internal/platform/crypto/token.go#L34) | `func NewSigner(secretKey []byte, purpose string) *Signer` |
| [internal/platform/crypto/token.go:44](../internal/platform/crypto/token.go#L44) | `func (s *Signer) Sign(payload []byte, ttl time.Duration) string` |
| [internal/platform/crypto/token.go:52](../internal/platform/crypto/token.go#L52) | `func (s *Signer) Verify(value string) ([]byte, error)` |
| [internal/platform/crypto/token.go:74](../internal/platform/crypto/token.go#L74) | `func (s *Signer) mac(body string) []byte` |
| [internal/platform/events/broker.go:46](../internal/platform/events/broker.go#L46) | `func NewBroker(pool *pgxpool.Pool, log *slog.Logger) *Broker` |
| [internal/platform/events/broker.go:51](../internal/platform/events/broker.go#L51) | `func (b *Broker) Ready() <-chan struct` |
| [internal/platform/events/broker.go:54](../internal/platform/events/broker.go#L54) | `func (b *Broker) Run(ctx context.Context) error` |
| [internal/platform/events/broker.go:109](../internal/platform/events/broker.go#L109) | `func (b *Broker) publish(ref Ref)` |
| [internal/platform/events/broker.go:125](../internal/platform/events/broker.go#L125) | `func (b *Broker) resync()` |
| [internal/platform/events/broker.go:137](../internal/platform/events/broker.go#L137) | `func (b *Broker) Subscribe(workspaceID uuid.UUID) *Subscription` |
| [internal/platform/events/broker.go:148](../internal/platform/events/broker.go#L148) | `func (s *Subscription) Close()` |
| [internal/platform/events/events.go:61](../internal/platform/events/events.go#L61) | `func Emit(ctx context.Context, q *dbgen.Queries, c Change) error` |
| [internal/platform/httpx/cookies.go:21](../internal/platform/httpx/cookies.go#L21) | `func (c Cookies) Session(token string, ttl time.Duration, remember bool) http.Cookie` |
| [internal/platform/httpx/cookies.go:33](../internal/platform/httpx/cookies.go#L33) | `func (c Cookies) Guest(value string, ttl time.Duration) http.Cookie` |
| [internal/platform/httpx/cookies.go:41](../internal/platform/httpx/cookies.go#L41) | `func (c Cookies) OAuth(value string) http.Cookie` |
| [internal/platform/httpx/cookies.go:49](../internal/platform/httpx/cookies.go#L49) | `func (c Cookies) Clear(name, path string) http.Cookie` |
| [internal/platform/httpx/etag.go:13](../internal/platform/httpx/etag.go#L13) | `func ETag(version int32) string` |
| [internal/platform/httpx/etag.go:17](../internal/platform/httpx/etag.go#L17) | `func IfMatch(header string) (version int32, ok bool, err error)` |
| [internal/platform/httpx/etag.go:39](../internal/platform/httpx/etag.go#L39) | `func EncodeCursor(v any) *string` |
| [internal/platform/httpx/etag.go:45](../internal/platform/httpx/etag.go#L45) | `func DecodeCursor(cursor string, into any) error` |
| [internal/platform/httpx/middleware.go:28](../internal/platform/httpx/middleware.go#L28) | `func ClientFrom(ctx context.Context) Client` |
| [internal/platform/httpx/middleware.go:35](../internal/platform/httpx/middleware.go#L35) | `func RequestContext(trusted []netip.Prefix) func(http.Handler) http.Handler` |
| [internal/platform/httpx/middleware.go:53](../internal/platform/httpx/middleware.go#L53) | `func AccessLog(log *slog.Logger) func(http.Handler) http.Handler` |
| [internal/platform/httpx/middleware.go:72](../internal/platform/httpx/middleware.go#L72) | `func SameOrigin(publicURL string) func(http.Handler) http.Handler` |
| [internal/platform/httpx/middleware.go:101](../internal/platform/httpx/middleware.go#L101) | `func ParsePrefixes(cidrs []string) ([]netip.Prefix, error)` |
| [internal/platform/httpx/middleware.go:113](../internal/platform/httpx/middleware.go#L113) | `func clientIP(r *http.Request, trusted []netip.Prefix) netip.Addr` |
| [internal/platform/httpx/middleware.go:141](../internal/platform/httpx/middleware.go#L141) | `func inAny(ip netip.Addr, prefixes []netip.Prefix) bool` |
| [internal/platform/httpx/middleware.go:150](../internal/platform/httpx/middleware.go#L150) | `func origin(raw string) string` |
| [internal/platform/httpx/middleware.go:158](../internal/platform/httpx/middleware.go#L158) | `func hasOurCookie(r *http.Request) bool` |
| [internal/platform/httpx/middleware.go:167](../internal/platform/httpx/middleware.go#L167) | `func truncate(s string, n int) string` |
| [internal/platform/httpx/middleware.go:179](../internal/platform/httpx/middleware.go#L179) | `func (w *statusWriter) WriteHeader(code int)` |
| [internal/platform/httpx/middleware.go:182](../internal/platform/httpx/middleware.go#L182) | `func (w *statusWriter) Flush()` |
| [internal/platform/httpx/middleware.go:188](../internal/platform/httpx/middleware.go#L188) | `func (w *statusWriter) Unwrap() http.ResponseWriter` |
| [internal/platform/httpx/operation.go:18](../internal/platform/httpx/operation.go#L18) | `func Op(perm authz.Permission, method, path, operationID, tag, summary string, opts ...OpOption) huma.Operation` |
| [internal/platform/httpx/operation.go:47](../internal/platform/httpx/operation.go#L47) | `func Status(code int) OpOption` |
| [internal/platform/httpx/operation.go:50](../internal/platform/httpx/operation.go#L50) | `func Errors(codes ...int) OpOption` |
| [internal/platform/httpx/operation.go:55](../internal/platform/httpx/operation.go#L55) | `func Describe(text string) OpOption` |
| [internal/platform/httpx/problem.go:35](../internal/platform/httpx/problem.go#L35) | `func (p *Problem) Error() string` |
| [internal/platform/httpx/problem.go:36](../internal/platform/httpx/problem.go#L36) | `func (p *Problem) GetStatus() int` |
| [internal/platform/httpx/problem.go:37](../internal/platform/httpx/problem.go#L37) | `func (p *Problem) GetHeaders() http.Header` |
| [internal/platform/httpx/problem.go:38](../internal/platform/httpx/problem.go#L38) | `func (p *Problem) ContentType(ct string) string` |
| [internal/platform/httpx/problem.go:55](../internal/platform/httpx/problem.go#L55) | `func InstallErrors(log *slog.Logger)` |
| [internal/platform/httpx/problem.go:98](../internal/platform/httpx/problem.go#L98) | `func fromApp(e *apperr.Error) *Problem` |
| [internal/platform/httpx/problem.go:107](../internal/platform/httpx/problem.go#L107) | `func codeForStatus(status int) string` |
| [internal/platform/logging/logging.go:14](../internal/platform/logging/logging.go#L14) | `func New(env, level string) *slog.Logger` |
| [internal/platform/logging/logging.go:39](../internal/platform/logging/logging.go#L39) | `func With(ctx context.Context, attrs ...slog.Attr) context.Context` |
| [internal/platform/logging/logging.go:50](../internal/platform/logging/logging.go#L50) | `func (h *ctxHandler) Handle(ctx context.Context, r slog.Record) error` |
| [internal/platform/logging/logging.go:57](../internal/platform/logging/logging.go#L57) | `func (h *ctxHandler) WithAttrs(attrs []slog.Attr) slog.Handler` |
| [internal/platform/logging/logging.go:61](../internal/platform/logging/logging.go#L61) | `func (h *ctxHandler) WithGroup(name string) slog.Handler` |
| [internal/platform/migrate/migrate.go:25](../internal/platform/migrate/migrate.go#L25) | `func Up(ctx context.Context, ownerURL string, log *slog.Logger) error` |
| [internal/platform/migrate/migrate.go:72](../internal/platform/migrate/migrate.go#L72) | `func Status(ctx context.Context, ownerURL string) ([]*goose.MigrationStatus, error)` |
| [internal/platform/pg/errors.go:19](../internal/platform/pg/errors.go#L19) | `func IsNotFound(err error) bool` |
| [internal/platform/pg/errors.go:23](../internal/platform/pg/errors.go#L23) | `func UniqueViolation(err error) (constraint string, ok bool)` |
| [internal/platform/pg/errors.go:28](../internal/platform/pg/errors.go#L28) | `func ForeignKeyViolation(err error) (constraint string, ok bool)` |
| [internal/platform/pg/errors.go:33](../internal/platform/pg/errors.go#L33) | `func CheckViolation(err error) (constraint string, ok bool)` |
| [internal/platform/pg/errors.go:39](../internal/platform/pg/errors.go#L39) | `func TriggerError(err error) (message string, ok bool)` |
| [internal/platform/pg/errors.go:47](../internal/platform/pg/errors.go#L47) | `func constraintFor(err error, code string) (string, bool)` |
| [internal/platform/pg/pg.go:30](../internal/platform/pg/pg.go#L30) | `func Open(ctx context.Context, url string, maxConns int32) (*DB, error)` |
| [internal/platform/pg/pg.go:57](../internal/platform/pg/pg.go#L57) | `func (d *DB) Close()` |
| [internal/platform/pg/pg.go:61](../internal/platform/pg/pg.go#L61) | `func (d *DB) Q() *dbgen.Queries` |
| [internal/platform/pg/pg.go:65](../internal/platform/pg/pg.go#L65) | `func (d *DB) Tx(ctx context.Context, fn func(q *dbgen.Queries) error) error` |
| [internal/platform/pg/pg.go:74](../internal/platform/pg/pg.go#L74) | `func (d *DB) InWorkspace(ctx context.Context, workspaceID uuid.UUID, fn func(q *dbgen.Queries) error) error` |
| [internal/platform/settings/settings.go:19](../internal/platform/settings/settings.go#L19) | `func SessionTTL(ctx context.Context, q *dbgen.Queries, fallback time.Duration) (time.Duration, error)` |
| [internal/platform/settings/settings.go:37](../internal/platform/settings/settings.go#L37) | `func OAuthAutoLink(ctx context.Context, q *dbgen.Queries) (bool, error)` |
| [internal/platform/slug/slug.go:8](../internal/platform/slug/slug.go#L8) | `func Make(name string) string` |
| [internal/platform/testutil/api.go:34](../internal/platform/testutil/api.go#L34) | `func NewAPI(t testing.TB) *API` |
| [internal/platform/testutil/api.go:39](../internal/platform/testutil/api.go#L39) | `func NewAPIWithOAuth(t testing.TB, providers map[string]auth.OAuthProvider) *API` |
| [internal/platform/testutil/api.go:43](../internal/platform/testutil/api.go#L43) | `func newAPI(t testing.TB, providers map[string]auth.OAuthProvider) *API` |
| [internal/platform/testutil/api.go:93](../internal/platform/testutil/api.go#L93) | `func (a *API) Client() *Client` |
| [internal/platform/testutil/api.go:108](../internal/platform/testutil/api.go#L108) | `func (r *Response) JSON(t testing.TB, v any)` |
| [internal/platform/testutil/api.go:116](../internal/platform/testutil/api.go#L116) | `func (r *Response) Code() string` |
| [internal/platform/testutil/api.go:126](../internal/platform/testutil/api.go#L126) | `func (c *Client) Do(method, path string, body any, headers ...string) *Response` |
| [internal/platform/testutil/api.go:162](../internal/platform/testutil/api.go#L162) | `func (r *Response) Expect(t testing.TB, status int) *Response` |
| [internal/platform/testutil/api.go:174](../internal/platform/testutil/api.go#L174) | `func (a *API) Owner() *Client` |
| [internal/platform/testutil/api.go:183](../internal/platform/testutil/api.go#L183) | `func (a *API) Login(username, password string) *Client` |
| [internal/platform/testutil/api.go:193](../internal/platform/testutil/api.go#L193) | `func (a *API) CreateUser(username, role string) *Client` |
| [internal/platform/testutil/api.go:202](../internal/platform/testutil/api.go#L202) | `func (a *API) AddMember(wsID, userID uuid.UUID, role string)` |
| [internal/platform/testutil/api.go:208](../internal/platform/testutil/api.go#L208) | `func (a *API) ClientAs(role string) *Client` |
| [internal/platform/testutil/api.go:226](../internal/platform/testutil/api.go#L226) | `func (c *Client) Stream(ctx context.Context, path string, headers ...string) *http.Response` |
| [internal/platform/testutil/api.go:243](../internal/platform/testutil/api.go#L243) | `func (w testWriter) Write(p []byte) (int, error)` |
| [internal/platform/testutil/postgres.go:53](../internal/platform/testutil/postgres.go#L53) | `func Postgres(t testing.TB) *DB` |
| [internal/platform/testutil/postgres.go:91](../internal/platform/testutil/postgres.go#L91) | `func ensureTemplate(ctx context.Context, adminURL string) (string, error)` |
| [internal/platform/testutil/postgres.go:162](../internal/platform/testutil/postgres.go#L162) | `func migrationsHash() (string, error)` |
| [internal/platform/testutil/postgres.go:181](../internal/platform/testutil/postgres.go#L181) | `func withUser(raw, user, database string) string` |
| [internal/platform/testutil/postgres.go:193](../internal/platform/testutil/postgres.go#L193) | `func connect(t testing.TB, url string) *pgx.Conn` |
| [internal/platform/testutil/postgres.go:202](../internal/platform/testutil/postgres.go#L202) | `func randomHex(n int) string` |
| [internal/server/schemas.go:21](../internal/server/schemas.go#L21) | `func newWireRegistry() *wireRegistry` |
| [internal/server/schemas.go:25](../internal/server/schemas.go#L25) | `func (r *wireRegistry) MarshalJSON() ([]byte, error)` |
| [internal/server/schemas.go:27](../internal/server/schemas.go#L27) | `func (r *wireRegistry) Schema(t reflect.Type, allowRef bool, hint string) *huma.Schema` |
| [internal/server/schemas.go:45](../internal/server/schemas.go#L45) | `func allowUnknownProperties(s *huma.Schema)` |
| [internal/server/schemas.go:57](../internal/server/schemas.go#L57) | `func wireUUIDs(s *huma.Schema, t reflect.Type)` |
| [internal/server/schemas.go:78](../internal/server/schemas.go#L78) | `func nullable(s *huma.Schema)` |
| [internal/server/schemas.go:87](../internal/server/schemas.go#L87) | `func floatPtr(n float64) *float64` |
| [internal/server/schemas.go:88](../internal/server/schemas.go#L88) | `func intPtr(n int) *int` |
| [internal/server/schemas.go:90](../internal/server/schemas.go#L90) | `func wireMetadata(name string, s *huma.Schema)` |
| [internal/server/schemas.go:193](../internal/server/schemas.go#L193) | `func healthEnum() []any` |
| [internal/server/schemas.go:197](../internal/server/schemas.go#L197) | `func shapeEnum() []any` |
| [internal/server/schemas.go:201](../internal/server/schemas.go#L201) | `func canvasSettingsSchema() *huma.Schema` |
| [internal/server/schemas.go:212](../internal/server/schemas.go#L212) | `func publishClientSchemas(api huma.API)` |
| [internal/server/server.go:51](../internal/server/server.go#L51) | `func New(d Deps) (http.Handler, error)` |
| [internal/server/server.go:89](../internal/server/server.go#L89) | `func problem(status int, code, detail string) http.HandlerFunc` |
| [internal/server/server.go:99](../internal/server/server.go#L99) | `func features(d Deps) []featureHandler` |
| [internal/server/server.go:145](../internal/server/server.go#L145) | `func apiConfig(dev bool) huma.Config` |
| [internal/server/server.go:176](../internal/server/server.go#L176) | `func OpenAPI() *huma.OpenAPI` |
| [internal/setup/handler.go:19](../internal/setup/handler.go#L19) | `func NewHandler(svc *Service) *Handler` |
| [internal/setup/handler.go:22](../internal/setup/handler.go#L22) | `func (h *Handler) Register(api huma.API)` |
| [internal/setup/handler.go:36](../internal/setup/handler.go#L36) | `func (h *Handler) status(ctx context.Context, _ *struct` |
| [internal/setup/handler.go:53](../internal/setup/handler.go#L53) | `func (h *Handler) complete(ctx context.Context, in *completeInput) (*completeOutput, error)` |
| [internal/setup/service.go:37](../internal/setup/service.go#L37) | `func NewService(db *pg.DB, sessions SessionStarter) *Service` |
| [internal/setup/service.go:42](../internal/setup/service.go#L42) | `func (s *Service) Status(ctx context.Context) (SetupStatus, error)` |
| [internal/setup/service.go:52](../internal/setup/service.go#L52) | `func (s *Service) Complete(ctx context.Context, req SetupRequest) (apimodel.Account, http.Cookie, error)` |
| [internal/setup/service.go:122](../internal/setup/service.go#L122) | `func optional(s string) *string` |
| [internal/share/handler.go:17](../internal/share/handler.go#L17) | `func NewHandler(svc *Service) *Handler` |
| [internal/share/handler.go:18](../internal/share/handler.go#L18) | `func (h *Handler) Register(api huma.API)` |
| [internal/share/handler.go:36](../internal/share/handler.go#L36) | `func (h *Handler) list(ctx context.Context, in *CanvasInput) (*listOutput, error)` |
| [internal/share/handler.go:52](../internal/share/handler.go#L52) | `func (h *Handler) create(ctx context.Context, in *createInput) (*createOutput, error)` |
| [internal/share/handler.go:72](../internal/share/handler.go#L72) | `func (h *Handler) update(ctx context.Context, in *updateInput) (*updateOutput, error)` |
| [internal/share/handler.go:80](../internal/share/handler.go#L80) | `func (h *Handler) revoke(ctx context.Context, in *LinkInput) (*struct` |
| [internal/share/handler.go:91](../internal/share/handler.go#L91) | `func (h *Handler) info(ctx context.Context, in *TokenInput) (*infoOutput, error)` |
| [internal/share/handler.go:109](../internal/share/handler.go#L109) | `func (h *Handler) unlock(ctx context.Context, in *unlockInput) (*unlockOutput, error)` |
| [internal/share/handler.go:126](../internal/share/handler.go#L126) | `func (in *viewInput) Resolve(ctx huma.Context) []error` |
| [internal/share/handler.go:136](../internal/share/handler.go#L136) | `func (h *Handler) view(ctx context.Context, in *viewInput) (*viewOutput, error)` |
| [internal/share/limiter.go:22](../internal/share/limiter.go#L22) | `func newLimiter() *limiter` |
| [internal/share/limiter.go:23](../internal/share/limiter.go#L23) | `func (l *limiter) allow(link, ip string) int` |
| [internal/share/limiter.go:49](../internal/share/limiter.go#L49) | `func refill(b bucket, now time.Time, capacity float64, period time.Duration) (bucket, int)` |
| [internal/share/model.go:44](../internal/share/model.go#L44) | `func (p *ShareLinkUpdate) UnmarshalJSON(data []byte) error` |
| [internal/share/model.go:72](../internal/share/model.go#L72) | `func fromRow(row dbgen.ShareLink) ShareLink` |
| [internal/share/service.go:37](../internal/share/service.go#L37) | `func NewService(db *pg.DB, az *authz.Resolver, cfg *config.Config, reader CanvasReader) *Service` |
| [internal/share/service.go:41](../internal/share/service.go#L41) | `func (s *Service) List(ctx context.Context, wsID, canvasID uuid.UUID) (ShareLinkPage, error)` |
| [internal/share/service.go:64](../internal/share/service.go#L64) | `func (s *Service) Create(ctx context.Context, wsID, canvasID uuid.UUID, in ShareLinkCreate) (ShareLinkCreated, error)` |
| [internal/share/service.go:99](../internal/share/service.go#L99) | `func (s *Service) Update(ctx context.Context, wsID, id uuid.UUID, in ShareLinkUpdate) (ShareLink, error)` |
| [internal/share/service.go:144](../internal/share/service.go#L144) | `func (s *Service) Revoke(ctx context.Context, wsID, id uuid.UUID) error` |
| [internal/share/service.go:166](../internal/share/service.go#L166) | `func (s *Service) active(ctx context.Context, token string) (dbgen.ShareLink, error)` |
| [internal/share/service.go:174](../internal/share/service.go#L174) | `func (s *Service) Info(ctx context.Context, token string) (ShareInfo, error)` |
| [internal/share/service.go:187](../internal/share/service.go#L187) | `func (s *Service) Unlock(ctx context.Context, token string, in ShareUnlock) (http.Cookie, error)` |
| [internal/share/service.go:228](../internal/share/service.go#L228) | `func (s *Service) View(ctx context.Context, token, cookie string) (apimodel.CanvasView, error)` |
| [internal/worker/worker.go:23](../internal/worker/worker.go#L23) | `func Run(ctx context.Context, cfg *config.Config, db *pg.DB, log *slog.Logger) error` |
| [internal/worker/worker.go:57](../internal/worker/worker.go#L57) | `func (cleanupSessionsArgs) Kind() string` |
| [internal/worker/worker.go:65](../internal/worker/worker.go#L65) | `func (w *cleanupSessionsWorker) Work(ctx context.Context, _ *river.Job[cleanupSessionsArgs]) error` |
| [internal/worker/worker.go:76](../internal/worker/worker.go#L76) | `func (pruneEventsArgs) Kind() string` |
| [internal/worker/worker.go:84](../internal/worker/worker.go#L84) | `func (w *pruneEventsWorker) Work(ctx context.Context, _ *river.Job[pruneEventsArgs]) error` |
| [internal/workspace/feature.go:33](../internal/workspace/feature.go#L33) | `func NewService(db *pg.DB, az *authz.Resolver) *Service` |
| [internal/workspace/feature.go:42](../internal/workspace/feature.go#L42) | `func NewHandler(svc *Service) *Handler` |
| [internal/workspace/feature.go:45](../internal/workspace/feature.go#L45) | `func (h *Handler) Register(api huma.API)` |
| [internal/workspace/feature.go:62](../internal/workspace/feature.go#L62) | `func (h *Handler) list(ctx context.Context, _ *struct` |
| [internal/workspace/feature.go:78](../internal/workspace/feature.go#L78) | `func (h *Handler) create(ctx context.Context, in *createInput) (*workspaceOutput, error)` |
| [internal/workspace/feature.go:90](../internal/workspace/feature.go#L90) | `func (h *Handler) get(ctx context.Context, in *workspaceInput) (*workspaceOutput, error)` |
| [internal/workspace/feature.go:104](../internal/workspace/feature.go#L104) | `func (h *Handler) update(ctx context.Context, in *updateInput) (*workspaceOutput, error)` |
| [internal/workspace/feature.go:125](../internal/workspace/feature.go#L125) | `func (h *Handler) delete(ctx context.Context, in *deleteInput) (*struct` |
| [internal/workspace/model.go:48](../internal/workspace/model.go#L48) | `func fromRow(row dbgen.Workspace, member Membership) (Workspace, error)` |
| [internal/workspace/service.go:19](../internal/workspace/service.go#L19) | `func (s *Service) List(ctx context.Context) (Page, error)` |
| [internal/workspace/service.go:43](../internal/workspace/service.go#L43) | `func (s *Service) Create(ctx context.Context, req WorkspaceCreate) (Workspace, error)` |
| [internal/workspace/service.go:76](../internal/workspace/service.go#L76) | `func (s *Service) Get(ctx context.Context, id uuid.UUID) (Workspace, error)` |
| [internal/workspace/service.go:91](../internal/workspace/service.go#L91) | `func (s *Service) Update(ctx context.Context, id uuid.UUID, req WorkspaceUpdate, match *int32) (Workspace, error)` |
| [internal/workspace/service.go:150](../internal/workspace/service.go#L150) | `func (s *Service) Delete(ctx context.Context, id uuid.UUID, match *int32) error` |
| [internal/workspace/service.go:182](../internal/workspace/service.go#L182) | `func workspaceError(err error) error` |
