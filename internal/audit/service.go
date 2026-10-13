// Package audit reads the append-only workspace audit log (permissions.md §3).
package audit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

type Service struct {
	db    *pg.DB
	authz *authz.Resolver
}

func NewService(db *pg.DB, az *authz.Resolver) *Service { return &Service{db: db, authz: az} }

type Actor struct {
	Type        string     `json:"type" enum:"user,share_link,system,workflow"`
	ID          *uuid.UUID `json:"id"`
	DisplayName *string    `json:"displayName"`
}
type (
	Target struct {
		Type string    `json:"type"`
		ID   uuid.UUID `json:"id"`
	}
	AuditEntry struct {
		ID        uuid.UUID      `json:"id"`
		Actor     Actor          `json:"actor"`
		Action    string         `json:"action"`
		Target    *Target        `json:"target"`
		Metadata  map[string]any `json:"metadata"`
		IP        *string        `json:"ip"`
		CreatedAt time.Time      `json:"createdAt"`
	}
)

type (
	AuditPage struct {
		Items      []AuditEntry `json:"items"`
		NextCursor *string      `json:"nextCursor" required:"false"`
	}
	cursor struct {
		Time time.Time
		ID   uuid.UUID
	}
)

func (s *Service) List(ctx context.Context, in listInput) (AuditPage, error) {
	if _, err := s.authz.Actor(ctx, in.WsID, authz.AuditList); err != nil {
		return AuditPage{}, err
	}
	params := dbgen.ListAuditEntriesParams{WorkspaceID: &in.WsID, Action: in.Action, PageLimit: int32(in.Limit + 1)}
	for _, filter := range []struct {
		raw string
		dst **uuid.UUID
	}{{in.ActorID, &params.ActorID}, {in.TargetID, &params.TargetID}} {
		if filter.raw != "" {
			id, err := uuid.Parse(filter.raw)
			if err != nil {
				return AuditPage{}, apperr.Invalid("validation", "Invalid audit filter ID.")
			}
			*filter.dst = &id
		}
	}
	if in.Since != "" {
		since, err := time.Parse(time.RFC3339Nano, in.Since)
		if err != nil {
			return AuditPage{}, apperr.Invalid("validation", "Invalid since timestamp.")
		}
		params.Since = &since
	}
	if in.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(in.Cursor)
		var c cursor
		if err != nil || json.Unmarshal(raw, &c) != nil || c.ID == uuid.Nil || c.Time.IsZero() {
			return AuditPage{}, apperr.Invalid("validation", "Invalid audit cursor.")
		}
		params.BeforeTime = &c.Time
		params.BeforeID = &c.ID
	}
	page := AuditPage{Items: []AuditEntry{}}
	err := s.db.InWorkspace(ctx, in.WsID, func(q *dbgen.Queries) error {
		rows, err := q.ListAuditEntries(ctx, params)
		if err != nil {
			return err
		}
		if len(rows) > in.Limit {
			rows = rows[:in.Limit]
			last := rows[len(rows)-1]
			raw, _ := json.Marshal(cursor{Time: last.CreatedAt, ID: last.ID})
			next := base64.RawURLEncoding.EncodeToString(raw)
			page.NextCursor = &next
		}
		for _, r := range rows {
			entry := AuditEntry{ID: r.ID, Actor: Actor{Type: r.ActorType, ID: r.ActorID, DisplayName: r.ActorDisplayName}, Action: r.Action, Metadata: map[string]any{}, CreatedAt: r.CreatedAt}
			if r.TargetType != nil && r.TargetID != nil {
				entry.Target = &Target{Type: *r.TargetType, ID: *r.TargetID}
			}
			if r.Ip != nil {
				ip := r.Ip.String()
				entry.IP = &ip
			}
			if err := json.Unmarshal(r.Metadata, &entry.Metadata); err != nil {
				return err
			}
			page.Items = append(page.Items, entry)
		}
		return nil
	})
	return page, err
}
