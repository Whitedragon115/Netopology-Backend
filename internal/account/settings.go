package account

import (
	"context"
	"encoding/json"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/auditlog"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/settings"
)

func readSettings(ctx context.Context, q *dbgen.Queries) (InstanceSettings, error) {
	ttl := 168
	autoLink := true
	for _, setting := range []struct {
		key    string
		target any
	}{
		{settings.SessionTTLKey, &ttl}, {settings.OAuthAutoLinkKey, &autoLink},
	} {
		raw, err := q.GetInstanceSetting(ctx, setting.key)
		if err != nil {
			if !pg.IsNotFound(err) {
				return InstanceSettings{}, err
			}
			continue
		}
		if err := json.Unmarshal(raw, setting.target); err != nil {
			return InstanceSettings{}, err
		}
	}
	return InstanceSettings{SessionTtlHours: &ttl, OauthAutoLinkByEmail: &autoLink}, nil
}

func (s *Service) GetInstanceSettings(ctx context.Context) (InstanceSettings, error) {
	if _, err := authz.RequireInstanceStaff(ctx); err != nil {
		return InstanceSettings{}, err
	}
	return readSettings(ctx, s.db.Q())
}

func (s *Service) UpdateInstanceSettings(ctx context.Context, in InstanceSettings) (InstanceSettings, error) {
	p, err := authz.RequireInstanceStaff(ctx)
	if err != nil {
		return InstanceSettings{}, err
	}
	if in.SessionTtlHours == nil && in.OauthAutoLinkByEmail == nil {
		return InstanceSettings{}, apperr.Invalid(apperr.CodeValidation, "Change at least one setting.")
	}
	var out InstanceSettings
	err = s.db.Tx(ctx, func(q *dbgen.Queries) error {
		fields := []string{}
		if in.SessionTtlHours != nil {
			raw, _ := json.Marshal(*in.SessionTtlHours)
			if err := q.UpsertInstanceSetting(ctx, dbgen.UpsertInstanceSettingParams{
				Key: settings.SessionTTLKey, Value: raw, UpdatedBy: &p.UserID,
			}); err != nil {
				return err
			}
			fields = append(fields, settings.SessionTTLKey)
		}
		if in.OauthAutoLinkByEmail != nil {
			raw, _ := json.Marshal(*in.OauthAutoLinkByEmail)
			if err := q.UpsertInstanceSetting(ctx, dbgen.UpsertInstanceSettingParams{
				Key: settings.OAuthAutoLinkKey, Value: raw, UpdatedBy: &p.UserID,
			}); err != nil {
				return err
			}
			fields = append(fields, settings.OAuthAutoLinkKey)
		}
		if err := auditlog.Write(ctx, q, auditlog.Entry{
			Action:     authz.InstanceSettingsWrite,
			TargetType: "instance_settings", Metadata: map[string]any{"fields": fields},
		}); err != nil {
			return err
		}
		var err error
		out, err = readSettings(ctx, q)
		return err
	})
	return out, err
}
