// Package auditlog writes audit entries. Every successful MUTATION writes exactly one entry, in
// the same transaction as the change (so a rolled-back change leaves no entry). Reads are not
// audited. The action is the operation's permission key.
//
// Never put secrets in Metadata (passwords, tokens, credential values). Changed field NAMES are
// fine; changed values only when they are not sensitive.
package auditlog

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/httpx"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

type Entry struct {
	WorkspaceID *uuid.UUID // nil for instance-level actions (users, settings, sign-in)
	Action      authz.Permission
	TargetType  string // "node", "canvas", "user", ...
	TargetID    *uuid.UUID
	Metadata    map[string]any
	// ActorID overrides the actor for actions without a session yet (login, setup, OAuth).
	ActorID *uuid.UUID
}

// Write records e using the caller from ctx (user, guest or system).
func Write(ctx context.Context, q *dbgen.Queries, e Entry) error {
	p := authz.PrincipalFrom(ctx)
	actorType, actorID := "system", (*uuid.UUID)(nil)
	switch {
	case e.ActorID != nil:
		actorType, actorID = "user", e.ActorID
	case p.SignedIn():
		id := p.UserID
		actorType, actorID = "user", &id
	case p.Guest != nil:
		id := p.Guest.ShareLinkID
		actorType, actorID = "share_link", &id
	}
	meta := e.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	var targetType *string
	if e.TargetType != "" {
		targetType = &e.TargetType
	}
	c := httpx.ClientFrom(ctx)
	params := dbgen.InsertAuditEntryParams{
		WorkspaceID: e.WorkspaceID,
		ActorType:   actorType,
		ActorID:     actorID,
		Action:      string(e.Action),
		TargetType:  targetType,
		TargetID:    e.TargetID,
		Metadata:    raw,
	}
	if c.IP.IsValid() {
		ip := c.IP
		params.Ip = &ip
	}
	if c.UserAgent != "" {
		ua := c.UserAgent
		params.UserAgent = &ua
	}
	return q.InsertAuditEntry(ctx, params)
}

// Ptr is a small helper for the optional uuid fields.
func Ptr(id uuid.UUID) *uuid.UUID { return &id }
