package drive

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type AuditEntry struct {
	ID         int64  `json:"id"`
	Timestamp  int64  `json:"ts"`
	Actor      string `json:"actor"`
	Action     string `json:"action"`
	TargetKind string `json:"target_kind,omitempty"`
	TargetID   string `json:"target_id,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

func (s *Service) WriteAudit(ctx context.Context, actor, action, targetKind, targetID string, detail any) {
	encoded := ""
	if detail != nil {
		if data, err := json.Marshal(detail); err == nil {
			encoded = string(data)
		}
	}
	if actor == "" {
		actor = "system"
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO audit_log (ts, actor, action, target_kind, target_id, detail) VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''))`, time.Now().Unix(), actor, action, targetKind, targetID, encoded)
}

func (s *Service) ListAudit(ctx context.Context, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, ts, actor, action, COALESCE(target_kind, ''), COALESCE(target_id, ''), COALESCE(detail, '') FROM audit_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("đọc audit log: %w", err)
	}
	defer rows.Close()
	entries := make([]AuditEntry, 0)
	for rows.Next() {
		var entry AuditEntry
		if err := rows.Scan(&entry.ID, &entry.Timestamp, &entry.Actor, &entry.Action, &entry.TargetKind, &entry.TargetID, &entry.Detail); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Service) PurgeAudit(ctx context.Context, olderThanSeconds int64) error {
	if olderThanSeconds <= 0 {
		return nil
	}
	cutoff := time.Now().Unix() - olderThanSeconds
	_, err := s.db.ExecContext(ctx, `DELETE FROM audit_log WHERE ts < ?`, cutoff)
	return err
}

var _ sql.Result = nil
