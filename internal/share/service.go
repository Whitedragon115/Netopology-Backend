// Package share manages read-only guest links and their signed unlock cookies.
package share

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/apimodel"
	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/auditlog"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/config"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/crypto"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

type (
	CanvasReader interface {
		ReadForGuest(context.Context, *dbgen.Queries) (apimodel.CanvasView, error)
	}
	Service struct {
		db      *pg.DB
		authz   *authz.Resolver
		cfg     *config.Config
		signer  *crypto.Signer
		reader  CanvasReader
		limiter *limiter
	}
)

func NewService(db *pg.DB, az *authz.Resolver, cfg *config.Config, reader CanvasReader) *Service {
	return &Service{db: db, authz: az, cfg: cfg, signer: crypto.NewSigner(cfg.SecretKey, "share"), reader: reader, limiter: newLimiter()}
}

func (s *Service) List(ctx context.Context, wsID, canvasID uuid.UUID) (ShareLinkPage, error) {
	if _, err := s.authz.Actor(ctx, wsID, authz.ShareLinkList); err != nil {
		return ShareLinkPage{}, err
	}
	out := ShareLinkPage{Items: []ShareLink{}}
	err := s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		if _, err := q.GetCanvasRow(ctx, dbgen.GetCanvasRowParams{WorkspaceID: wsID, ID: canvasID}); pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		} else if err != nil {
			return err
		}
		rows, err := q.ListShareLinks(ctx, dbgen.ListShareLinksParams{WorkspaceID: wsID, CanvasID: canvasID})
		if err != nil {
			return err
		}
		for _, r := range rows {
			out.Items = append(out.Items, fromRow(r))
		}
		return nil
	})
	return out, err
}

func (s *Service) Create(ctx context.Context, wsID, canvasID uuid.UUID, in ShareLinkCreate) (ShareLinkCreated, error) {
	actor, err := s.authz.Actor(ctx, wsID, authz.ShareLinkCreate)
	if err != nil {
		return ShareLinkCreated{}, err
	}
	token, err := crypto.NewToken()
	if err != nil {
		return ShareLinkCreated{}, err
	}
	var hash *string
	if in.Password != nil {
		h, err := crypto.HashPassword(*in.Password)
		if err != nil {
			return ShareLinkCreated{}, err
		}
		hash = &h
	}
	var out ShareLinkCreated
	err = s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		if _, err := q.GetCanvasRow(ctx, dbgen.GetCanvasRowParams{WorkspaceID: wsID, ID: canvasID}); pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		} else if err != nil {
			return err
		}
		userID := actor.UserID()
		row, err := q.CreateShareLink(ctx, dbgen.CreateShareLinkParams{WorkspaceID: wsID, CanvasID: canvasID, Label: in.Label, TokenHash: crypto.HashToken(token), PasswordHash: hash, ExpiresAt: in.ExpiresAt, CreatedBy: &userID})
		if err != nil {
			return err
		}
		out = ShareLinkCreated{ShareLink: fromRow(row), Token: token, URL: s.cfg.PublicURL + "/share/" + token}
		return auditlog.Write(ctx, q, auditlog.Entry{WorkspaceID: &wsID, Action: authz.ShareLinkCreate, TargetType: "share_link", TargetID: &row.ID})
	})
	return out, err
}

func (s *Service) Update(ctx context.Context, wsID, id uuid.UUID, in ShareLinkUpdate) (ShareLink, error) {
	if _, err := s.authz.Actor(ctx, wsID, authz.ShareLinkUpdate); err != nil {
		return ShareLink{}, err
	}
	if in.Label == nil && !in.PasswordSet && !in.ExpirySet {
		return ShareLink{}, apperr.Invalid("validation", "Provide at least one share link field.")
	}
	var hash *string
	if in.PasswordSet && in.Password != nil {
		h, err := crypto.HashPassword(*in.Password)
		if err != nil {
			return ShareLink{}, err
		}
		hash = &h
	}
	var out ShareLink
	err := s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		row, err := q.GetShareLink(ctx, dbgen.GetShareLinkParams{WorkspaceID: wsID, ID: id})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if in.Label != nil {
			row.Label = *in.Label
		}
		if in.ExpirySet {
			row.ExpiresAt = in.ExpiresAt
		}
		changed := int64(0)
		if in.PasswordSet {
			row.PasswordHash = hash
			changed = 1
		}
		row, err = q.UpdateShareLink(ctx, dbgen.UpdateShareLinkParams{WorkspaceID: wsID, ID: id, Label: row.Label, PasswordHash: row.PasswordHash, ExpiresAt: row.ExpiresAt, PasswordChanged: changed})
		if err != nil {
			return err
		}
		out = fromRow(row)
		return auditlog.Write(ctx, q, auditlog.Entry{WorkspaceID: &wsID, Action: authz.ShareLinkUpdate, TargetType: "share_link", TargetID: &id})
	})
	return out, err
}

func (s *Service) Revoke(ctx context.Context, wsID, id uuid.UUID) error {
	if _, err := s.authz.Actor(ctx, wsID, authz.ShareLinkRevoke); err != nil {
		return err
	}
	return s.db.InWorkspace(ctx, wsID, func(q *dbgen.Queries) error {
		row, err := q.GetShareLink(ctx, dbgen.GetShareLinkParams{WorkspaceID: wsID, ID: id})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if row.RevokedAt != nil {
			return nil
		}
		if _, err = q.RevokeShareLink(ctx, dbgen.RevokeShareLinkParams{WorkspaceID: wsID, ID: id}); err != nil {
			return err
		}
		return auditlog.Write(ctx, q, auditlog.Entry{WorkspaceID: &wsID, Action: authz.ShareLinkRevoke, TargetType: "share_link", TargetID: &id})
	})
}

func (s *Service) active(ctx context.Context, token string) (dbgen.ShareLink, error) {
	row, err := s.db.Q().GetActiveShareByToken(ctx, crypto.HashToken(token))
	if pg.IsNotFound(err) {
		return row, apperr.ErrNotFound()
	}
	return row, err
}

func (s *Service) Info(ctx context.Context, token string) (ShareInfo, error) {
	row, err := s.active(ctx, token)
	if err != nil {
		return ShareInfo{}, err
	}
	return ShareInfo{RequiresPassword: row.PasswordHash != nil}, nil
}

type grantCookie struct {
	ShareLinkID     uuid.UUID `json:"shareLinkId"`
	PasswordVersion int64     `json:"passwordVersion"`
}

func (s *Service) Unlock(ctx context.Context, token string, in ShareUnlock) (http.Cookie, error) {
	row, err := s.active(ctx, token)
	if err != nil {
		return http.Cookie{}, err
	}
	if retry := s.limiter.allow(row.ID.String(), httpx.ClientFrom(ctx).IP.String()); retry > 0 {
		return http.Cookie{}, apperr.RateLimited(retry)
	}
	if row.PasswordHash != nil {
		valid, err := crypto.VerifyPassword(in.Password, *row.PasswordHash)
		if err != nil {
			return http.Cookie{}, err
		}
		if !valid {
			return http.Cookie{}, apperr.Unauthenticated("share.password_required", "Incorrect share password.")
		}
	}
	grant := &authz.GuestGrant{ShareLinkID: row.ID, WorkspaceID: row.WorkspaceID, CanvasID: row.CanvasID}
	guestCtx := authz.WithPrincipal(ctx, &authz.Principal{Guest: grant})
	err = s.db.InWorkspace(guestCtx, row.WorkspaceID, func(q *dbgen.Queries) error {
		used, err := q.RecordShareUse(ctx, dbgen.RecordShareUseParams{WorkspaceID: row.WorkspaceID, ID: row.ID, PasswordVersion: row.PasswordVersion})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		row = used
		return auditlog.Write(guestCtx, q, auditlog.Entry{WorkspaceID: &row.WorkspaceID, Action: authz.ShareUnlock, TargetType: "share_link", TargetID: &row.ID})
	})
	if err != nil {
		return http.Cookie{}, err
	}
	ttl := 24 * time.Hour
	if row.ExpiresAt != nil {
		ttl = min(ttl, time.Until(*row.ExpiresAt))
	}
	payload, _ := json.Marshal(grantCookie{ShareLinkID: row.ID, PasswordVersion: row.PasswordVersion})
	return (httpx.Cookies{Secure: s.cfg.CookieSecure}).Guest(s.signer.Sign(payload, ttl), ttl), nil
}

func (s *Service) View(ctx context.Context, token, cookie string) (apimodel.CanvasView, error) {
	row, err := s.active(ctx, token)
	if err != nil {
		return apimodel.CanvasView{}, err
	}
	raw, err := s.signer.Verify(cookie)
	var unlocked grantCookie
	if err != nil || json.Unmarshal(raw, &unlocked) != nil || unlocked.ShareLinkID != row.ID || unlocked.PasswordVersion != row.PasswordVersion {
		return apimodel.CanvasView{}, apperr.Unauthenticated("share.password_required", "Unlock this share link first.")
	}
	guestCtx := authz.WithPrincipal(ctx, &authz.Principal{Guest: &authz.GuestGrant{ShareLinkID: row.ID, WorkspaceID: row.WorkspaceID, CanvasID: row.CanvasID}})
	var out apimodel.CanvasView
	err = s.db.InWorkspace(guestCtx, row.WorkspaceID, func(q *dbgen.Queries) error {
		current, err := q.LockActiveShareLink(guestCtx, dbgen.LockActiveShareLinkParams{WorkspaceID: row.WorkspaceID, ID: row.ID})
		if pg.IsNotFound(err) {
			return apperr.ErrNotFound()
		}
		if err != nil {
			return err
		}
		if current.PasswordVersion != unlocked.PasswordVersion {
			return apperr.Unauthenticated("share.password_required", "Unlock this share link first.")
		}
		out, err = s.reader.ReadForGuest(guestCtx, q)
		return err
	})
	return out, err
}
