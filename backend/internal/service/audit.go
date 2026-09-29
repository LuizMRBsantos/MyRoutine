package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/myroutine/backend/internal/appctx"
)

// Audit log ("livro de ocorrências"): who did what, when, from which IP and
// browser. Rows about a person use their user_id, so they come with the LGPD
// export and are anonymized (no owner, no IP, no user agent) when the account
// is deleted.

// Audit actions. Keep them stable: they are the log's vocabulary.
const (
	AuditLogin                = "auth.login"
	AuditLoginFailed          = "auth.login_failed"
	AuditRegister             = "auth.register"
	AuditPasswordChanged      = "auth.password_changed"
	AuditPasswordReset        = "auth.password_reset"
	AuditSessionReuseDetected = "auth.session_reuse_detected"
	AuditInviteCreated        = "admin.invite_created"
	AuditInviteRevoked        = "admin.invite_revoked"
	AuditResetLinkCreated     = "admin.password_reset_link_created"
	AuditAccountExported      = "account.exported"
	AuditAccountDeleted       = "account.deleted"
)

// auditEvent is one audit row. UserID may be empty (e.g. a failed login for
// an email with no account). EntityID must be a UUID or empty.
type auditEvent struct {
	UserID     string
	Action     string
	EntityType string
	EntityID   string
	Metadata   map[string]any
	// Anonymous skips IP and user agent (used for the account-deletion row).
	Anonymous bool
}

// recordAudit writes an audit row through db, which may be a transaction so
// the row lands (or not) together with the action it describes. IP and user
// agent come from the request context (see middleware.RequestMeta).
func recordAudit(ctx context.Context, db dbExecer, ev auditEvent) error {
	var ip, ua any
	if meta := appctx.RequestMetaFrom(ctx); !ev.Anonymous {
		if meta.IP != "" {
			ip = meta.IP
		}
		if meta.UserAgent != "" {
			ua = meta.UserAgent
		}
	}

	var metadata any
	if len(ev.Metadata) > 0 {
		raw, err := json.Marshal(ev.Metadata)
		if err != nil {
			return fmt.Errorf("encoding audit metadata: %w", err)
		}
		metadata = string(raw)
	}

	_, err := db.Exec(ctx,
		`INSERT INTO audit_logs (user_id, action, entity_type, entity_id, ip_address, user_agent, metadata)
		 VALUES (NULLIF($1, '')::uuid, $2, NULLIF($3, ''), NULLIF($4, '')::uuid, $5::inet, $6, $7::jsonb)`,
		ev.UserID, ev.Action, ev.EntityType, ev.EntityID, ip, ua, metadata,
	)
	if err != nil {
		return fmt.Errorf("recording audit %s: %w", ev.Action, err)
	}
	return nil
}
